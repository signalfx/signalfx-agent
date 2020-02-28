import os
import re
import subprocess
import threading
import time
from contextlib import contextmanager
from pathlib import Path

import docker
import requests
from tests.helpers import fake_backend
from tests.helpers.util import (
    get_docker_client,
    get_host_ip,
    pull_from_reader_in_background,
    retry,
    retry_on_ebadf,
    run_container,
)
from tests.paths import REPO_ROOT_DIR

PACKAGING_DIR = REPO_ROOT_DIR / "packaging"
DEPLOYMENTS_DIR = REPO_ROOT_DIR / "deployments"
INSTALLER_PATH = DEPLOYMENTS_DIR / "installer/install.sh"
RPM_OUTPUT_DIR = PACKAGING_DIR / "rpm/output/x86_64"
DEB_OUTPUT_DIR = PACKAGING_DIR / "deb/output"
DOCKERFILES_DIR = Path(__file__).parent.joinpath("images").resolve()
WIN_AGENT_LATEST_URL = "https://dl.signalfx.com/windows/{stage}/zip/latest/latest.txt"
WIN_AGENT_PATH = r"C:\Program Files\SignalFx\SignalFxAgent\bin\signalfx-agent.exe"
WIN_REPO_ROOT_DIR = os.path.realpath(os.path.join(os.path.dirname(os.path.realpath(__file__)), "..", ".."))
WIN_INSTALLER_PATH = os.path.join(WIN_REPO_ROOT_DIR, "deployments", "installer", "install.ps1")
WIN_UNINSTALLER_PATH = os.path.join(WIN_REPO_ROOT_DIR, "scripts", "windows", "uninstall-agent.ps1")

INIT_SYSV = "sysv"
INIT_UPSTART = "upstart"
INIT_SYSTEMD = "systemd"

AGENT_YAML_PATH = "/etc/signalfx/agent.yaml"
PIDFILE_PATH = "/var/run/signalfx-agent.pid"

BASIC_CONFIG = """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/cpu
  - type: collectd/uptime
"""


def build_base_image(name, path=DOCKERFILES_DIR, dockerfile=None, buildargs=None):
    client = get_docker_client()
    dockerfile = dockerfile or Path(path) / f"Dockerfile.{name}"
    image, _ = client.images.build(
        path=str(path), dockerfile=str(dockerfile), pull=True, rm=True, forcerm=True, buildargs=buildargs
    )

    return image.id


LOG_COMMAND = {
    INIT_SYSV: "cat /var/log/signalfx-agent.log",
    INIT_UPSTART: "cat /var/log/signalfx-agent.log",
    INIT_SYSTEMD: "journalctl -u signalfx-agent",
}


def get_agent_logs(container, init_system):
    try:
        _, output = container.exec_run(LOG_COMMAND[init_system])
    except docker.errors.APIError as e:
        print("Error getting agent logs: %s" % e)
        return ""
    return output


def get_deb_package_to_test():
    return get_package_to_test(DEB_OUTPUT_DIR, "deb")


def get_rpm_package_to_test():
    return get_package_to_test(RPM_OUTPUT_DIR, "rpm")


def get_package_to_test(output_dir, extension):
    pkgs = list(Path(output_dir).glob(f"*.{extension}"))
    if not pkgs:
        raise AssertionError(f"No .{extension} files found in {output_dir}")

    if len(pkgs) > 1:
        raise AssertionError(f"More than one .{extension} file found in {output_dir}")

    return pkgs[0]


# Run an HTTPS proxy inside the container with socat so that our fake backend
# doesn't have to worry about HTTPS.  The cert file must be trusted by the
# container running the agent.
# This is pretty hacky but docker makes it hard to communicate from a container
# back to the host machine (and we don't want to use the host network stack in
# the container due to init systems).  The idea is to bind mount a shared
# folder from the test host to the container that two socat instances use to
# communicate using a file to make the bytes flow between the HTTPS proxy and
# the fake backend.
@contextmanager
def socat_https_proxy(container, target_host, target_port, source_host, bind_addr):
    cert = "/%s.cert" % source_host
    key = "/%s.key" % source_host

    socat_bin = DOCKERFILES_DIR / "socat"
    stopped = False
    socket_path = "/tmp/scratch/%s-%s" % (source_host, container.id[:12])

    # Keep the socat instance in the container running across container
    # restarts
    def keep_running_in_container(cont, sock):
        while not stopped:
            try:
                cont.exec_run(
                    [
                        "socat",
                        "-v",
                        "OPENSSL-LISTEN:443,cert=%s,key=%s,verify=0,bind=%s,fork" % (cert, key, bind_addr),
                        "UNIX-CONNECT:%s" % sock,
                    ]
                )
            except docker.errors.APIError:
                print("socat died, restarting...")
                time.sleep(0.1)

    threading.Thread(target=keep_running_in_container, args=(container, socket_path)).start()

    proc = retry_on_ebadf(
        lambda: subprocess.Popen(
            [socat_bin, "-v", "UNIX-LISTEN:%s,fork" % socket_path, "TCP4:%s:%d" % (target_host, target_port)],
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            close_fds=False,
        )
    )()

    get_local_out = pull_from_reader_in_background(proc.stdout)

    try:
        yield
    finally:
        stopped = True
        # The socat instance in the container will die with the container
        proc.kill()
        print(get_local_out())


@contextmanager
def run_init_system_image(
    base_image,
    with_socat=True,
    path=DOCKERFILES_DIR,
    dockerfile=None,
    ingest_host="ingest.us0.signalfx.com",  # Whatever value is used here needs a self-signed cert in ./images/certs/
    api_host="api.us0.signalfx.com",  # Whatever value is used here needs a self-signed cert in ./images/certs/
    command=None,
    buildargs=None,
):  # pylint: disable=too-many-arguments
    image_id = retry(lambda: build_base_image(base_image, path, dockerfile, buildargs), docker.errors.BuildError)
    print("Image ID: %s" % image_id)
    if with_socat:
        backend_ip = "127.0.0.1"
    else:
        backend_ip = get_host_ip()
    with fake_backend.start(ip_addr=backend_ip) as backend:
        container_options = {
            # Init systems running in the container want permissions
            "privileged": True,
            "volumes": {
                "/sys/fs/cgroup": {"bind": "/sys/fs/cgroup", "mode": "ro"},
                "/tmp/scratch": {"bind": "/tmp/scratch", "mode": "rw"},
            },
            "extra_hosts": {
                # Socat will be running on localhost to forward requests to
                # these hosts to the fake backend
                ingest_host: backend.ingest_host,
                api_host: backend.api_host,
            },
        }

        if command:
            container_options["command"] = command

        with run_container(image_id, wait_for_ip=True, **container_options) as cont:
            if with_socat:
                # Proxy the backend calls through a fake HTTPS endpoint so that we
                # don't have to change the default configuration default by the
                # package.  The base_image used should trust the self-signed certs
                # default in the images dir so that the agent doesn't throw TLS
                # verification errors.
                with socat_https_proxy(
                    cont, backend.ingest_host, backend.ingest_port, ingest_host, "127.0.0.1"
                ), socat_https_proxy(cont, backend.api_host, backend.api_port, api_host, "127.0.0.2"):
                    yield [cont, backend]
            else:
                yield [cont, backend]


@retry_on_ebadf
def is_agent_running_as_non_root(container):
    code, output = container.exec_run("pgrep -u signalfx-agent signalfx-agent")
    print("pgrep check: %s" % output)
    return code == 0


@retry_on_ebadf
def get_agent_version(cont):
    code, output = cont.exec_run("signalfx-agent -version")
    output = output.decode("utf-8").strip()
    assert code == 0, "command 'signalfx-agent -version' failed:\n%s" % output
    match = re.match("^.+?: (.+)?,", output)
    assert match and match.group(1).strip(), "failed to parse agent version from command output:\n%s" % output
    return match.group(1).strip()


def run_win_command(cmd, returncodes=None, shell=True, **kwargs):
    if returncodes is None:
        returncodes = [0]
    print('running "%s" ...' % cmd)
    proc = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, shell=shell, close_fds=False, **kwargs)
    output = proc.stdout.decode("utf-8")
    if returncodes:
        assert proc.returncode in returncodes, output
    print(output)
    return proc


def get_win_agent_version(agent_path=WIN_AGENT_PATH):
    proc = run_win_command([agent_path, "-version"])
    output = proc.stdout.decode("utf-8")
    match = re.match("^.+?: (.+)?,", output)
    assert match and match.group(1).strip(), "failed to parse agent version from command output:\n%s" % output
    return match.group(1).strip()


def running_in_azure_pipelines():
    return os.environ.get("AZURE_HTTP_USER_AGENT") is not None


def has_choco():
    return run_win_command("choco --version", []).returncode == 0


def uninstall_win_agent():
    run_win_command(f'powershell.exe "{WIN_UNINSTALLER_PATH}"')


def get_latest_win_agent_version(stage="release"):
    return requests.get(WIN_AGENT_LATEST_URL.format(stage=stage)).text.strip()


def import_old_key(cont, distro_type):
    if distro_type == "deb":
        cmd = (
            "bash -ec 'curl -o /etc/apt/trusted.gpg.d/signalfx.gpg https://dl.signalfx.com/debian.gpg && "
            "sleep 2 &&"
            "apt-key add /etc/apt/trusted.gpg.d/signalfx.gpg'"
        )
    else:
        cmd = "rpm --import https://dl.signalfx.com/yum-rpm.key"
    code, output = cont.exec_run(cmd, tty=True)
    assert code == 0, output.decode("utf-8")


def assert_old_key_removed(cont, distro_type):
    if distro_type == "deb":
        code, output = cont.exec_run("apt-key list")
        assert "5ae495f6" not in output.decode("utf-8").lower(), "old key still exists!"
        code, output = cont.exec_run("test -f /etc/apt/trusted.gpg.d/signalfx.gpg")
        assert code != 0, "old key file still exists!"
    else:
        code, output = cont.exec_run("rpm -q gpg-pubkey-098acf3b-55a5351a")
        assert code != 0, "old key still exists!"

import os
import subprocess
import tarfile
import threading
import time
from contextlib import contextmanager
from io import BytesIO
from pathlib import Path

import docker

from tests.helpers import fake_backend
from tests.helpers.util import get_docker_client, get_host_ip, retry, run_container
from tests.paths import REPO_ROOT_DIR

PACKAGING_DIR = REPO_ROOT_DIR / "packaging"
DEPLOYMENTS_DIR = REPO_ROOT_DIR / "deployments"
INSTALLER_PATH = DEPLOYMENTS_DIR / "installer/install.sh"
RPM_OUTPUT_DIR = PACKAGING_DIR / "rpm/output/x86_64"
DEB_OUTPUT_DIR = PACKAGING_DIR / "deb/output"
DOCKERFILES_DIR = Path(__file__).parent.joinpath("images").resolve()

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


def build_base_image(name, path=DOCKERFILES_DIR, dockerfile=None):
    client = get_docker_client()
    dockerfile = dockerfile or Path(path) / f"Dockerfile.{name}"
    image, _ = client.images.build(path=str(path), dockerfile=str(dockerfile), pull=True, rm=True, forcerm=True)

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

    proc = subprocess.Popen(
        [socat_bin, "-v", "UNIX-LISTEN:%s,fork" % socket_path, "TCP4:%s:%d" % (target_host, target_port)],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )

    def read_out(_p):
        while True:
            read_bytes = _p.stdout.read()
            if not read_bytes:
                return
            print(read_bytes)

    threading.Thread(target=read_out, args=(proc,)).start()

    try:
        yield
    finally:
        stopped = True
        # The socat instance in the container will die with the container
        proc.kill()


def copy_file_content_into_container(content, container, target_path):
    copy_file_object_into_container(
        BytesIO(content.encode("utf-8")), container, target_path, size=len(content.encode("utf-8"))
    )


# This is more convoluted that it should be but seems to be the simplest way in
# the face of docker-in-docker environments where volume bind mounting is hard.
def copy_file_object_into_container(fd, container, target_path, size=None):
    tario = BytesIO()
    tar = tarfile.TarFile(fileobj=tario, mode="w")

    info = tarfile.TarInfo(name=target_path)
    if size is None:
        size = os.fstat(fd.fileno()).st_size
    info.size = size

    tar.addfile(info, fd)

    tar.close()

    container.put_archive("/", tario.getvalue())
    # Apparently when the above `put_archive` call returns, the file isn't
    # necessarily fully written in the container, so wait a bit to ensure it
    # is.
    time.sleep(2)


def copy_file_into_container(path, container, target_path):
    with open(path, "rb") as fd:
        copy_file_object_into_container(fd, container, target_path)


@contextmanager
def run_init_system_image(
    base_image,
    with_socat=True,
    path=DOCKERFILES_DIR,
    dockerfile=None,
    ingest_host="ingest.us0.signalfx.com",  # Whatever value is used here needs a self-signed cert in ./images/certs/
    api_host="api.us0.signalfx.com",  # Whatever value is used here needs a self-signed cert in ./images/certs/
    command=None,
):
    image_id = retry(lambda: build_base_image(base_image, path, dockerfile), docker.errors.BuildError)
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
                # don't have to change the default configuration included by the
                # package.  The base_image used should trust the self-signed certs
                # included in the images dir so that the agent doesn't throw TLS
                # verification errors.
                with socat_https_proxy(
                    cont, backend.ingest_host, backend.ingest_port, ingest_host, "127.0.0.1"
                ), socat_https_proxy(cont, backend.api_host, backend.api_port, api_host, "127.0.0.2"):
                    yield [cont, backend]
            else:
                yield [cont, backend]


def is_agent_running_as_non_root(container):
    code, output = container.exec_run("pgrep -u signalfx-agent signalfx-agent")
    print("pgrep check: %s" % output)
    return code == 0


def path_exists_in_container(container, path):
    code, _ = container.exec_run("test -e %s" % path)
    return code == 0


def get_container_file_content(container, path):
    assert path_exists_in_container(container, path), "File %s does not exist!" % path
    return container.exec_run("cat %s" % path)[1].decode("utf-8")

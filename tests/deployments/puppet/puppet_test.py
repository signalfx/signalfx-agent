import os
import shutil
import string
import subprocess
import sys
import tempfile
from functools import partial as p
from pathlib import Path

import pytest

from tests.helpers.agent import ensure_fake_backend
from tests.helpers.assertions import has_datapoint_with_dim, has_datapoint_with_metric_name
from tests.helpers.formatting import print_dp_or_event
from tests.helpers.util import print_lines, wait_for, copy_file_into_container
from tests.packaging.common import (
    INIT_SYSTEMD,
    INIT_UPSTART,
    get_agent_logs,
    get_agent_version,
    get_win_agent_version,
    is_agent_running_as_non_root,
    run_init_system_image,
)
from tests.paths import REPO_ROOT_DIR

pytestmark = [pytest.mark.puppet, pytest.mark.deployment]

DOCKERFILES_DIR = Path(__file__).parent.joinpath("images").resolve()

APT_MODULE_VERSION = "7.0.0"

DEB_DISTROS = [
    ("debian-8-jessie", INIT_SYSTEMD),
    ("debian-9-stretch", INIT_SYSTEMD),
    ("ubuntu1404", INIT_UPSTART),
    ("ubuntu1604", INIT_SYSTEMD),
    ("ubuntu1804", INIT_SYSTEMD),
]

RPM_DISTROS = [
    ("amazonlinux1", INIT_UPSTART),
    ("amazonlinux2", INIT_SYSTEMD),
    ("centos6", INIT_UPSTART),
    ("centos7", INIT_SYSTEMD),
]

CONFIG = string.Template(
    """
class { signalfx_agent:
    package_stage => '$stage',
    agent_version => '$version',
    config => {
        signalFxAccessToken => 'testing123',
        ingestUrl => '$ingest_url',
        apiUrl => '$api_url',
        observers => [
            { type => "host" },
        ],
        monitors => [
            $monitors
        ],
    }
}
"""
)

STAGE = "final"
INITIAL_VERSION = "4.7.5"
UPGRADE_VERSION = "4.7.6"


def get_config(backend, monitors, version, stage):
    if not version:
        version = ""
    return CONFIG.substitute(
        version=version, stage=stage, ingest_url=backend.ingest_url, api_url=backend.api_url, monitors=monitors
    )


def run_puppet_agent(cont, backend, monitors, agent_version=None, stage=STAGE):
    with tempfile.NamedTemporaryFile(mode="w+") as fd:
        fd.write(get_config(backend, monitors, agent_version, stage))
        fd.flush()
        copy_file_into_container(fd.name, cont, "/root/agent.pp")
    code, output = cont.exec_run("puppet apply /root/agent.pp")
    assert code in (0, 2), output.decode("utf-8")
    print_lines(output)
    installed_version = get_agent_version(cont)
    if agent_version:
        assert installed_version == agent_version, "installed agent version is '%s', expected '%s'" % (
            installed_version,
            agent_version,
        )
    return installed_version


@pytest.mark.parametrize(
    "base_image,init_system",
    [pytest.param(distro, init, marks=pytest.mark.deb) for distro, init in DEB_DISTROS]
    + [pytest.param(distro, init, marks=pytest.mark.rpm) for distro, init in RPM_DISTROS],
)
def test_puppet(base_image, init_system):
    dockerfile = os.path.join(DOCKERFILES_DIR, "Dockerfile.%s" % base_image)
    with run_init_system_image(base_image, path=REPO_ROOT_DIR, dockerfile=dockerfile, with_socat=False) as [
        cont,
        backend,
    ]:
        if (base_image, init_system) in DEB_DISTROS:
            code, output = cont.exec_run(f"puppet module install puppetlabs-apt --version {APT_MODULE_VERSION}")
            assert code == 0, output.decode("utf-8")
            print_lines(output)
        try:
            monitors = '{ type => "host-metadata" },'
            run_puppet_agent(cont, backend, monitors, INITIAL_VERSION, STAGE)
            assert is_agent_running_as_non_root(cont), "Agent is not running as non-root user"
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"
            run_puppet_agent(cont, backend, monitors, UPGRADE_VERSION, STAGE)
            backend.reset_datapoints()
            assert is_agent_running_as_non_root(cont), "Agent is not running as non-root user"
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"
            run_puppet_agent(cont, backend, monitors, INITIAL_VERSION, STAGE)
            backend.reset_datapoints()
            assert is_agent_running_as_non_root(cont), "Agent is not running as non-root user"
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"
            monitors = '{ type => "internal-metrics" },'
            run_puppet_agent(cont, backend, monitors, INITIAL_VERSION, STAGE)
            backend.reset_datapoints()
            assert is_agent_running_as_non_root(cont), "Agent is not running as non-root user"
            assert wait_for(
                p(has_datapoint_with_metric_name, backend, "sfxagent.datapoints_sent")
            ), "Didn't get internal metric datapoints"
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))


def run_win_command(cmd, returncodes=None):
    if returncodes is None:
        returncodes = [0]
    print('running "%s" ...' % cmd)
    proc = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, shell=True)
    output = proc.stdout.decode("utf-8")
    if returncodes:
        assert proc.returncode in returncodes, output
    print(output)
    return proc


def get_win_puppet_version():
    return run_win_command("puppet --version").stdout.decode("utf-8").strip()


def run_win_puppet_agent(backend, monitors, agent_version=None, stage=STAGE):
    manifest_path = "agent.pp"
    with open(manifest_path, "w+") as fd:
        fd.write(get_config(backend, monitors, agent_version, stage))
    cmd = f"puppet apply {manifest_path}"
    run_win_command(cmd, [0, 2])
    installed_version = get_win_agent_version()
    if agent_version:
        assert installed_version == agent_version, "installed agent version is '%s', expected '%s'" % (
            installed_version,
            agent_version,
        )
    return installed_version


if sys.platform == "win32":
    WIN_PUPPET_VERSIONS = (
        ["5.0.0", "latest"] if os.environ.get("USERNAME") == "VssAdministrator" else [get_win_puppet_version()]
    )
else:
    WIN_PUPPET_VERSIONS = []
WIN_REPO_ROOT_DIR = os.path.realpath(os.path.join(os.path.dirname(os.path.realpath(__file__)), "..", "..", ".."))
WIN_UNINSTALL_SCRIPT_PATH = os.path.join(WIN_REPO_ROOT_DIR, "scripts", "windows", "uninstall-agent.ps1")
WIN_PUPPET_BIN_DIR = r"C:\Program Files\Puppet Labs\Puppet\bin"
WIN_PUPPET_MODULE_SRC_DIR = os.path.join(WIN_REPO_ROOT_DIR, "deployments", "puppet")
WIN_PUPPET_MODULE_DEST_DIR = r"C:\ProgramData\PuppetLabs\code\environments\production\modules\signalfx_agent"


@pytest.mark.windows_only
@pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows")
@pytest.mark.parametrize("puppet_version", WIN_PUPPET_VERSIONS)
def test_puppet_on_windows(puppet_version):
    run_win_command(f"powershell.exe '{WIN_UNINSTALL_SCRIPT_PATH}'")
    if os.environ.get("USERNAME") == "VssAdministrator":
        if puppet_version == "latest":
            run_win_command(f"choco upgrade -y -f puppet-agent")
        else:
            run_win_command(f"choco upgrade -y -f puppet-agent --version {puppet_version}")
        if WIN_PUPPET_BIN_DIR not in os.environ.get("PATH"):
            os.environ["PATH"] = WIN_PUPPET_BIN_DIR + ";" + os.environ.get("PATH")
    if os.path.isdir(WIN_PUPPET_MODULE_DEST_DIR):
        shutil.rmtree(WIN_PUPPET_MODULE_DEST_DIR)
    shutil.copytree(WIN_PUPPET_MODULE_SRC_DIR, WIN_PUPPET_MODULE_DEST_DIR)
    run_win_command("puppet module install puppet-archive")
    run_win_command("puppet module install puppetlabs-powershell")
    with ensure_fake_backend() as backend:
        try:
            monitors = '{ type => "host-metadata" },'
            run_win_puppet_agent(backend, monitors, INITIAL_VERSION, STAGE)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"
            run_win_puppet_agent(backend, monitors, UPGRADE_VERSION, STAGE)
            backend.reset_datapoints()
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"
            run_win_puppet_agent(backend, monitors, INITIAL_VERSION, STAGE)
            backend.reset_datapoints()
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"
            monitors = '{ type => "internal-metrics" },'
            run_win_puppet_agent(backend, monitors, INITIAL_VERSION, STAGE)
            backend.reset_datapoints()
            assert wait_for(
                p(has_datapoint_with_metric_name, backend, "sfxagent.datapoints_sent")
            ), "Didn't get internal metric datapoints"
        finally:
            print("\nDatapoints received:")
            for dp in backend.datapoints:
                print_dp_or_event(dp)
            print("\nEvents received:")
            for event in backend.events:
                print_dp_or_event(event)
            print(f"\nDimensions set: {backend.dims}")

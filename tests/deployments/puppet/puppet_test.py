import os
import shutil
import string
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
    WIN_REPO_ROOT_DIR,
    get_agent_logs,
    get_agent_version,
    get_win_agent_version,
    has_choco,
    is_agent_running_as_non_root,
    run_init_system_image,
    run_win_command,
    uninstall_win_agent,
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

LINUX_PUPPET_RELEASES = os.environ.get("PUPPET_RELEASES", "5,latest").split(",")
WIN_PUPPET_VERSIONS = os.environ.get("PUPPET_VERSIONS", "5.0.0,latest").split(",")

STAGE = "final"
INITIAL_VERSION = "4.7.5"
UPGRADE_VERSION = "4.7.6"

WIN_PUPPET_BIN_DIR = r"C:\Program Files\Puppet Labs\Puppet\bin"
WIN_PUPPET_MODULE_SRC_DIR = os.path.join(WIN_REPO_ROOT_DIR, "deployments", "puppet")
WIN_PUPPET_MODULE_DEST_DIR = r"C:\ProgramData\PuppetLabs\code\environments\production\modules\signalfx_agent"


def get_config(backend, monitors, version, stage):
    if not version:
        version = ""
    return CONFIG.substitute(
        version=version, stage=stage, ingest_url=backend.ingest_url, api_url=backend.api_url, monitors=monitors
    )


def run_puppet_agent(cont, backend, monitors, agent_version, stage):
    with tempfile.NamedTemporaryFile(mode="w+") as fd:
        fd.write(get_config(backend, monitors, agent_version, stage))
        fd.flush()
        copy_file_into_container(fd.name, cont, "/root/agent.pp")
    code, output = cont.exec_run("puppet apply /root/agent.pp")
    assert code in (0, 2), output.decode("utf-8")
    print_lines(output)
    installed_version = get_agent_version(cont)
    assert installed_version == agent_version, "installed agent version is '%s', expected '%s'" % (
        installed_version,
        agent_version,
    )
    assert is_agent_running_as_non_root(cont), "Agent is not running as non-root user"
    backend.reset_datapoints()


@pytest.mark.parametrize(
    "base_image,init_system",
    [pytest.param(distro, init, marks=pytest.mark.deb) for distro, init in DEB_DISTROS]
    + [pytest.param(distro, init, marks=pytest.mark.rpm) for distro, init in RPM_DISTROS],
)
@pytest.mark.parametrize("puppet_release", LINUX_PUPPET_RELEASES)
def test_puppet(base_image, init_system, puppet_release):
    if puppet_release == "latest":
        buildargs = {"PUPPET_RELEASE": ""}
    else:
        buildargs = {"PUPPET_RELEASE": puppet_release}
    opts = {
        "path": REPO_ROOT_DIR,
        "dockerfile": os.path.join(DOCKERFILES_DIR, "Dockerfile.%s" % base_image),
        "with_socat": False,
        "buildargs": buildargs,
    }
    with run_init_system_image(base_image, **opts) as [cont, backend]:
        if (base_image, init_system) in DEB_DISTROS:
            code, output = cont.exec_run(f"puppet module install puppetlabs-apt --version {APT_MODULE_VERSION}")
            assert code == 0, output.decode("utf-8")
            print_lines(output)
        try:
            monitors = '{ type => "host-metadata" },'
            run_puppet_agent(cont, backend, monitors, INITIAL_VERSION, STAGE)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            # upgrade agent
            run_puppet_agent(cont, backend, monitors, UPGRADE_VERSION, STAGE)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            # downgrade agent
            run_puppet_agent(cont, backend, monitors, INITIAL_VERSION, STAGE)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            # change agent config
            monitors = '{ type => "internal-metrics" },'
            run_puppet_agent(cont, backend, monitors, INITIAL_VERSION, STAGE)
            assert wait_for(
                p(has_datapoint_with_metric_name, backend, "sfxagent.datapoints_sent")
            ), "Didn't get internal metric datapoints"
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))


def run_win_puppet_agent(backend, monitors, agent_version, stage):
    manifest_path = "agent.pp"
    config = get_config(backend, monitors, agent_version, stage)
    print(config)
    with open(manifest_path, "w+") as fd:
        fd.write(config)
    cmd = f"puppet apply {manifest_path}"
    run_win_command(cmd, [0, 2])
    installed_version = get_win_agent_version()
    assert installed_version == agent_version, "installed agent version is '%s', expected '%s'" % (
        installed_version,
        agent_version,
    )
    backend.reset_datapoints()


def run_win_puppet_setup(puppet_version):
    assert has_choco(), "choco not installed!"
    uninstall_win_agent()
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


@pytest.mark.windows_only
@pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows")
@pytest.mark.parametrize("puppet_version", WIN_PUPPET_VERSIONS)
def test_puppet_on_windows(puppet_version):
    run_win_puppet_setup(puppet_version)
    with ensure_fake_backend() as backend:
        try:
            monitors = '{ type => "host-metadata" },'
            run_win_puppet_agent(backend, monitors, INITIAL_VERSION, STAGE)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            # upgrade agent
            run_win_puppet_agent(backend, monitors, UPGRADE_VERSION, STAGE)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            # downgrade agent
            run_win_puppet_agent(backend, monitors, INITIAL_VERSION, STAGE)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            # change agent config
            monitors = '{ type => "internal-metrics" },'
            run_win_puppet_agent(backend, monitors, INITIAL_VERSION, STAGE)
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

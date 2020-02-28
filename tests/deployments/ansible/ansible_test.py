import os
import re
import tempfile
from functools import partial as p
from pathlib import Path

import pytest
import yaml

from tests.helpers.assertions import has_datapoint_with_dim, has_datapoint_with_metric_name
from tests.helpers.util import print_lines, wait_for, copy_file_into_container
from tests.packaging.common import (
    INIT_SYSTEMD,
    INIT_UPSTART,
    assert_old_key_removed,
    get_agent_logs,
    get_agent_version,
    import_old_key,
    is_agent_running_as_non_root,
    run_init_system_image,
)
from tests.paths import REPO_ROOT_DIR

pytestmark = [pytest.mark.ansible, pytest.mark.deployment]

DOCKERFILES_DIR = Path(__file__).parent.joinpath("images").resolve()

DEB_DISTROS = [
    ("debian-8-jessie", INIT_SYSTEMD),
    ("debian-9-stretch", INIT_SYSTEMD),
    ("ubuntu1404", INIT_UPSTART),
    ("ubuntu1604", INIT_SYSTEMD),
    ("ubuntu1804", INIT_SYSTEMD),
]

RPM_DISTROS = [("amazonlinux2", INIT_SYSTEMD), ("centos7", INIT_SYSTEMD), ("centos8", INIT_SYSTEMD)]

CONFIG = """
sfx_package_stage: null
sfx_version: null
sfx_agent_config:
  signalFxAccessToken: testing123
  ingestUrl: null
  apiUrl: null
  intervalSeconds: 1
  observers:
    - type: host
  monitors: null
"""

PLAYBOOK_DEST_DIR = "/opt/playbook"
INVENTORY_DEST_PATH = os.path.join(PLAYBOOK_DEST_DIR, "inventory.ini")
CONFIG_DEST_PATH = os.path.join(PLAYBOOK_DEST_DIR, "config.yml")
PLAYBOOK_DEST_PATH = os.path.join(PLAYBOOK_DEST_DIR, "playbook.yml")
ANSIBLE_CMD = f"ansible-playbook -vvvv -i {INVENTORY_DEST_PATH} -e @{CONFIG_DEST_PATH} {PLAYBOOK_DEST_PATH}"

ANSIBLE_VERSIONS = os.environ.get("ANSIBLE_VERSIONS", "2.5.0,latest").split(",")
STAGE = os.environ.get("STAGE", "release")
INITIAL_VERSION = os.environ.get("INITIAL_VERSION", "4.14.0")
UPGRADE_VERSION = os.environ.get("UPGRADE_VERSION", "4.15.0")


def get_config(backend, monitors, agent_version, stage):
    config_yaml = yaml.safe_load(CONFIG)
    config_yaml["sfx_package_stage"] = stage
    config_yaml["sfx_version"] = agent_version + "-1"
    config_yaml["sfx_agent_config"]["ingestUrl"] = backend.ingest_url
    config_yaml["sfx_agent_config"]["apiUrl"] = backend.api_url
    config_yaml["sfx_agent_config"]["monitors"] = monitors
    return yaml.dump(config_yaml)


def run_ansible(cont, backend, monitors, agent_version, stage):
    with tempfile.NamedTemporaryFile(mode="w+") as fd:
        config_yaml = get_config(backend, monitors, agent_version, stage)
        print(config_yaml)
        fd.write(config_yaml)
        fd.flush()
        copy_file_into_container(fd.name, cont, CONFIG_DEST_PATH)
    code, output = cont.exec_run(ANSIBLE_CMD)
    assert code == 0, output.decode("utf-8")
    print_lines(output)
    installed_version = get_agent_version(cont).replace("~", "-")
    agent_version = re.sub(r"-\d+$", "", agent_version).replace("~", "-")
    assert installed_version == agent_version, "installed agent version is '%s', expected '%s'" % (
        installed_version,
        agent_version,
    )
    assert is_agent_running_as_non_root(cont), "Agent is not running as non-root user"


@pytest.mark.parametrize(
    "base_image,init_system",
    [pytest.param(distro, init, marks=pytest.mark.deb) for distro, init in DEB_DISTROS]
    + [pytest.param(distro, init, marks=pytest.mark.rpm) for distro, init in RPM_DISTROS],
)
@pytest.mark.parametrize("ansible_version", ANSIBLE_VERSIONS)
def test_ansible(base_image, init_system, ansible_version):
    if (base_image, init_system) in DEB_DISTROS:
        distro_type = "deb"
    else:
        distro_type = "rpm"
    if base_image == "centos8" and ansible_version != "latest" and tuple(ansible_version.split(".")) < ("2", "8", "1"):
        pytest.skip(f"ansible {ansible_version} not supported on {base_image}")
    buildargs = {"ANSIBLE_VERSION": ""}
    if ansible_version != "latest":
        buildargs = {"ANSIBLE_VERSION": f"=={ansible_version}"}
    opts = {
        "path": REPO_ROOT_DIR,
        "dockerfile": DOCKERFILES_DIR / f"Dockerfile.{base_image}",
        "buildargs": buildargs,
        "with_socat": False,
    }
    with run_init_system_image(base_image, **opts) as [cont, backend]:
        import_old_key(cont, distro_type)
        try:
            monitors = [{"type": "host-metadata"}]
            run_ansible(cont, backend, monitors, INITIAL_VERSION, STAGE)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            assert_old_key_removed(cont, distro_type)

            if UPGRADE_VERSION:
                # upgrade agent
                run_ansible(cont, backend, monitors, UPGRADE_VERSION, STAGE)
                backend.reset_datapoints()
                assert wait_for(
                    p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
                ), "Datapoints didn't come through"

                # downgrade agent
                run_ansible(cont, backend, monitors, INITIAL_VERSION, STAGE)
                backend.reset_datapoints()
                assert wait_for(
                    p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
                ), "Datapoints didn't come through"

            # change agent config
            monitors = [{"type": "internal-metrics"}]
            run_ansible(cont, backend, monitors, INITIAL_VERSION, STAGE)
            backend.reset_datapoints()
            assert wait_for(
                p(has_datapoint_with_metric_name, backend, "sfxagent.datapoints_sent")
            ), "Didn't get internal metric datapoints"
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))

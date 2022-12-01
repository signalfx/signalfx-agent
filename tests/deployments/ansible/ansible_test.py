import os
import re
import tempfile
from functools import partial as p
from pathlib import Path

import pytest
import yaml

from tests.helpers.assertions import has_datapoint_with_dim, has_datapoint_with_metric_name
from tests.helpers.util import copy_file_into_container, print_lines, wait_for
from tests.packaging.common import (
    INIT_SYSTEMD,
    INIT_UPSTART,
    assert_old_key_removed,
    get_agent_logs,
    get_agent_version,
    import_old_key,
    is_agent_running_as_non_root,
    run_init_system_image,
    verify_override_files,
)
from tests.paths import REPO_ROOT_DIR

pytestmark = [pytest.mark.ansible, pytest.mark.deployment]

DOCKERFILES_DIR = Path(__file__).parent.joinpath("images").resolve()

DEB_DISTROS = [
    ("debian-9-stretch", INIT_SYSTEMD),
    ("ubuntu1604", INIT_SYSTEMD),
    ("ubuntu1804", INIT_SYSTEMD),
]

RPM_DISTROS = [
    ("amazonlinux1", INIT_UPSTART),
    ("amazonlinux2", INIT_SYSTEMD),
    ("centos7", INIT_SYSTEMD),
    ("centos8", INIT_SYSTEMD),
]

CONFIG = """
sfx_package_stage: null
sfx_version: null
sfx_service_user: null
sfx_service_group: null
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

ANSIBLE_VERSIONS = os.environ.get("ANSIBLE_VERSIONS", "3.0.0,latest").split(",")
STAGE = os.environ.get("STAGE", "release")
INITIAL_VERSION = os.environ.get("INITIAL_VERSION", "4.14.0")
UPGRADE_VERSION = os.environ.get("UPGRADE_VERSION", "5.1.0")


def get_config(backend, monitors, agent_version, stage, user):
    config_yaml = yaml.safe_load(CONFIG)
    config_yaml["sfx_package_stage"] = stage
    config_yaml["sfx_version"] = agent_version + "-1"
    config_yaml["sfx_service_user"] = user
    config_yaml["sfx_service_group"] = user
    config_yaml["sfx_agent_config"]["ingestUrl"] = backend.ingest_url
    config_yaml["sfx_agent_config"]["apiUrl"] = backend.api_url
    config_yaml["sfx_agent_config"]["monitors"] = monitors
    return yaml.dump(config_yaml)


def run_ansible(cont, init_system, backend, monitors, agent_version, stage, user="signalfx-agent"):
    with tempfile.NamedTemporaryFile(mode="w+") as fd:
        config_yaml = get_config(backend, monitors, agent_version, stage, user)
        print(config_yaml)
        fd.write(config_yaml)
        fd.flush()
        copy_file_into_container(fd.name, cont, CONFIG_DEST_PATH)
    code, output = cont.exec_run(ANSIBLE_CMD)
    assert code == 0, output.decode("utf-8")
    print_lines(output)
    verify_override_files(cont, init_system, user)
    installed_version = get_agent_version(cont).replace("~", "-")
    agent_version = re.sub(r"-\d+$", "", agent_version).replace("~", "-")
    assert installed_version == agent_version, "installed agent version is '%s', expected '%s'" % (
        installed_version,
        agent_version,
    )
    assert is_agent_running_as_non_root(cont, user=user), f"Agent is not running as {user} user"


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
            run_ansible(cont, init_system, backend, monitors, INITIAL_VERSION, STAGE)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            assert_old_key_removed(cont, distro_type)

            if UPGRADE_VERSION:
                # upgrade agent
                run_ansible(cont, init_system, backend, monitors, UPGRADE_VERSION, STAGE, user="test-user")
                backend.reset_datapoints()
                assert wait_for(
                    p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
                ), "Datapoints didn't come through"

                # downgrade agent
                run_ansible(cont, init_system, backend, monitors, INITIAL_VERSION, STAGE)
                backend.reset_datapoints()
                assert wait_for(
                    p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
                ), "Datapoints didn't come through"

            # change agent config
            monitors = [{"type": "internal-metrics"}]
            run_ansible(cont, init_system, backend, monitors, INITIAL_VERSION, STAGE)
            backend.reset_datapoints()
            assert wait_for(
                p(has_datapoint_with_metric_name, backend, "sfxagent.datapoints_sent")
            ), "Didn't get internal metric datapoints"
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))

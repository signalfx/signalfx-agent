import os
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
    verify_override_files,
)
from tests.paths import REPO_ROOT_DIR

pytestmark = [pytest.mark.salt, pytest.mark.deployment]

DOCKERFILES_DIR = Path(__file__).parent.joinpath("images").resolve()

DEB_DISTROS = [
    ("debian-9-stretch", INIT_SYSTEMD),
    ("ubuntu1804", INIT_SYSTEMD),
]

RPM_DISTROS = [
    ("amazonlinux1", INIT_UPSTART),
    ("amazonlinux2", INIT_SYSTEMD),
    ("centos7", INIT_SYSTEMD),
    ("centos8", INIT_SYSTEMD),
]

CONFIG = """
signalfx-agent:
  package_stage: null
  version: null
  service_user: null
  service_group: null
  conf:
    signalFxAccessToken: 'testing123'
    ingestUrl: null
    apiUrl: null
    intervalSeconds: 1
    observers:
      - type: host
    monitors: null
"""

PILLAR_PATH = "/srv/pillar/signalfx-agent.sls"
SALT_CMD = "salt-call --local state.apply"
STAGE = os.environ.get("STAGE", "release")
INITIAL_VERSION = os.environ.get("INITIAL_VERSION", "4.7.5")
UPGRADE_VERSION = os.environ.get("UPGRADE_VERSION", "5.1.0")


def get_config(backend, agent_version, monitors, stage, user, group=None):
    if group is None:
        group = user

    config_yaml = yaml.safe_load(CONFIG)
    config_yaml["signalfx-agent"]["package_stage"] = stage
    config_yaml["signalfx-agent"]["version"] = agent_version + "-1"
    config_yaml["signalfx-agent"]["service_user"] = user
    config_yaml["signalfx-agent"]["service_group"] = group
    config_yaml["signalfx-agent"]["conf"]["ingestUrl"] = backend.ingest_url
    config_yaml["signalfx-agent"]["conf"]["apiUrl"] = backend.api_url
    config_yaml["signalfx-agent"]["conf"]["monitors"] = monitors

    return yaml.dump(config_yaml)


def run_salt(cont, init_system, backend, agent_version, monitors, stage, user="signalfx-agent"):
    with tempfile.NamedTemporaryFile(mode="w+") as fd:
        config_yaml = get_config(backend, agent_version, monitors, stage, user)
        print(config_yaml)
        fd.write(config_yaml)
        fd.flush()
        copy_file_into_container(fd.name, cont, PILLAR_PATH)

    code, output = cont.exec_run(SALT_CMD)
    print_lines(output)
    assert code == 0, f"'{SALT_CMD}' failed"

    verify_override_files(cont, init_system, user)

    installed_version = get_agent_version(cont)
    assert installed_version == agent_version, "installed agent version is '%s', expected '%s'" % (
        installed_version,
        agent_version,
    )

    assert is_agent_running_as_non_root(cont, user), f"Agent is not running as {user} user"


@pytest.mark.parametrize(
    "base_image,init_system",
    [pytest.param(distro, init, marks=pytest.mark.deb) for distro, init in DEB_DISTROS]
    + [pytest.param(distro, init, marks=pytest.mark.rpm) for distro, init in RPM_DISTROS],
)
def test_salt(base_image, init_system):
    if (base_image, init_system) in DEB_DISTROS:
        distro_type = "deb"
    else:
        distro_type = "rpm"

    opts = {"path": REPO_ROOT_DIR, "dockerfile": DOCKERFILES_DIR / f"Dockerfile.{base_image}", "with_socat": False}
    with run_init_system_image(base_image, **opts) as [cont, backend]:
        try:
            import_old_key(cont, distro_type)

            monitors = [{"type": "host-metadata"}]
            run_salt(cont, init_system, backend, INITIAL_VERSION, monitors, STAGE)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            assert_old_key_removed(cont, distro_type)

            if UPGRADE_VERSION:
                # upgrade agent
                user = "signalfx-agent"
                if tuple(UPGRADE_VERSION.split(".")) >= ("5", "1", "0"):
                    user = "test-user"
                run_salt(cont, init_system, backend, UPGRADE_VERSION, monitors, STAGE, user)
                backend.reset_datapoints()
                assert wait_for(
                    p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
                ), "Datapoints didn't come through"

                # downgrade agent
                run_salt(cont, init_system, backend, INITIAL_VERSION, monitors, STAGE)
                backend.reset_datapoints()
                assert wait_for(
                    p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
                ), "Datapoints didn't come through"

            # change agent config
            monitors = [{"type": "internal-metrics"}]
            run_salt(cont, init_system, backend, INITIAL_VERSION, monitors, STAGE)
            backend.reset_datapoints()
            assert wait_for(
                p(has_datapoint_with_metric_name, backend, "sfxagent.datapoints_sent")
            ), "Didn't get internal metric datapoints"

        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))

# Tests of the chef cookbook

import json
import os
import tempfile
from functools import partial as p
from pathlib import Path

import pytest

from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.util import print_lines, wait_for
from tests.packaging.common import (
    INIT_SYSTEMD,
    INIT_UPSTART,
    get_agent_logs,
    get_agent_version,
    is_agent_running_as_non_root,
    run_init_system_image,
)
from tests.paths import REPO_ROOT_DIR

pytestmark = [pytest.mark.chef, pytest.mark.deployment]

ATTRIBUTES_JSON = """
{"signalfx_agent": {
  "package_stage": "%s",
  "agent_version": null,
  "conf": {
    "signalFxAccessToken": "testing",
    "ingestUrl": "https://ingest.us0.signalfx.com",
    "apiUrl": "https://api.us0.signalfx.com",
    "observers": [
      {"type": "host"}
    ],
    "monitors": [
      {"type": "host-metadata"}
    ]
  }
}}
"""
CHEF_CMD = "chef-client -z -o 'recipe[signalfx_agent::default]' -j {0}"
SCRIPT_DIR = Path(__file__).parent.resolve()
DOCKERFILES_DIR = SCRIPT_DIR / "images"

SUPPORTED_DISTROS = [
    ("debian-8-jessie", INIT_SYSTEMD),
    ("debian-9-stretch", INIT_SYSTEMD),
    ("ubuntu1404", INIT_UPSTART),
    ("ubuntu1604", INIT_SYSTEMD),
    ("ubuntu1804", INIT_SYSTEMD),
    ("amazonlinux1", INIT_UPSTART),
    ("amazonlinux2", INIT_SYSTEMD),
    ("centos6", INIT_UPSTART),
    ("centos7", INIT_SYSTEMD),
    ("opensuse15", INIT_SYSTEMD),
]

# allow CHEF_VERSIONS env var with comma-separated chef versions for test parameterization
CHEF_VERSIONS = os.environ.get("CHEF_VERSIONS", "latest").split(",")


def run_chef_client(cont, chef_version, agent_version=None, stage="final"):
    attributes = json.loads(ATTRIBUTES_JSON % stage)
    attributes["signalfx_agent"]["agent_version"] = agent_version
    print(attributes)
    with tempfile.NamedTemporaryFile(mode="w", dir="/tmp/scratch") as fd:
        fd.write(json.dumps(attributes))
        fd.flush()
        cmd = CHEF_CMD.format(fd.name)
        if chef_version == "latest" or int(chef_version.split(".")[0]) >= 15:
            cmd += " --chef-license accept-silent"
        print('running "%s" ...' % cmd)
        code, output = cont.exec_run(cmd)
        output = output.decode("utf-8").strip()
        assert code == 0, "failed to install agent:\n%s" % output
        print(output)
    assert is_agent_running_as_non_root(cont), "Agent is not running as non-root user"
    return get_agent_version(cont)


@pytest.mark.parametrize("base_image,init_system", SUPPORTED_DISTROS)
@pytest.mark.parametrize("chef_version", CHEF_VERSIONS)
def test_chef(base_image, init_system, chef_version):
    dockerfile = DOCKERFILES_DIR / f"Dockerfile.{base_image}"
    buildargs = {"CHEF_INSTALLER_ARGS": ""}
    if chef_version != "latest":
        buildargs["CHEF_INSTALLER_ARGS"] = f"-v {chef_version}"
    with run_init_system_image(base_image, path=REPO_ROOT_DIR, dockerfile=dockerfile, buildargs=buildargs) as [
        cont,
        backend,
    ]:
        try:
            # use rpm in test stage until the suse-supported rpm is released to final stage
            stage = "final" if "opensuse" not in base_image else "test"
            # install latest agent
            run_chef_client(cont, chef_version, stage=stage)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))


@pytest.mark.upgrade_downgrade
@pytest.mark.parametrize("base_image,init_system", SUPPORTED_DISTROS)
@pytest.mark.parametrize("chef_version", CHEF_VERSIONS)
def test_chef_upgrade_downgrade(base_image, init_system, chef_version):
    if "opensuse" in base_image:
        pytest.skip("not yet supported")
    dockerfile = DOCKERFILES_DIR / f"Dockerfile.{base_image}"
    buildargs = {"CHEF_INSTALLER_ARGS": ""}
    if chef_version != "latest":
        buildargs["CHEF_INSTALLER_ARGS"] = f"-v {chef_version}"
    with run_init_system_image(base_image, path=REPO_ROOT_DIR, dockerfile=dockerfile, buildargs=buildargs) as [
        cont,
        backend,
    ]:
        try:
            agent_version = run_chef_client(cont, chef_version, "4.1.1")
            assert agent_version == "4.1.1", "agent version is not 4.1.1: %s" % agent_version
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            # upgrade agent
            agent_version = run_chef_client(cont, chef_version, "4.2.0")
            assert agent_version == "4.2.0", "agent version is not 4.2.0: %s" % agent_version
            backend.reset_datapoints()
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            # downgrade agent for distros that support package downgrades
            if base_image not in ("debian-7-wheezy", "debian-8-jessie", "ubuntu1404"):
                agent_version = run_chef_client(cont, chef_version, "4.1.0")
                assert agent_version == "4.1.0", "agent version is not 4.1.0: %s" % agent_version
                backend.reset_datapoints()
                assert wait_for(
                    p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
                ), "Datapoints didn't come through"
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))

# Tests of the chef cookbook

import json
import os
import re
import tempfile
from functools import partial as p

import pytest

from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.util import print_lines, wait_for
from tests.packaging.common import (
    INIT_SYSTEMD,
    INIT_UPSTART,
    PROJECT_DIR,
    get_agent_logs,
    is_agent_running_as_non_root,
    run_init_system_image,
)

pytestmark = [pytest.mark.chef, pytest.mark.deployment]

ATTRIBUTES_PATH = os.path.join(PROJECT_DIR, "deployments/chef/example_attrs.json")
CHEF_CMD = "chef-client -z -o 'recipe[signalfx_agent::default]' -j {0}"
DOCKERFILES_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), "images"))

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
]


def get_agent_version(cont):
    code, output = cont.exec_run("signalfx-agent -version")
    output = output.decode("utf-8").strip()
    assert code == 0, "command 'signalfx-agent -version' failed:\n%s" % output
    match = re.match("^.+?: (.+)?,", output)
    assert match and match.group(1).strip(), "failed to parse agent version from command output:\n%s" % output
    return match.group(1).strip()


def run_chef_client(cont, agent_version=None):
    attributes = json.loads(open(ATTRIBUTES_PATH, "r").read())
    if agent_version:
        attributes["signalfx_agent"]["agent_version"] = agent_version
    attributes["signalfx_agent"]["conf"]["ingestUrl"] = "https://ingest.us0.signalfx.com"
    attributes["signalfx_agent"]["conf"]["apiUrl"] = "https://api.us0.signalfx.com"
    print(attributes)
    with tempfile.NamedTemporaryFile(mode="w", dir="/tmp/scratch") as fd:
        fd.write(json.dumps(attributes))
        fd.flush()
        print('running "%s" ...' % CHEF_CMD.format(fd.name))
        code, output = cont.exec_run(CHEF_CMD.format(fd.name))
        output = output.decode("utf-8").strip()
        assert code == 0, "failed to install agent:\n%s" % output
        print(output)
    assert is_agent_running_as_non_root(cont), "Agent is not running as non-root user"
    return get_agent_version(cont)


@pytest.mark.parametrize("base_image,init_system", SUPPORTED_DISTROS)
def test_chef(base_image, init_system):
    dockerfile = os.path.join(DOCKERFILES_DIR, "Dockerfile.%s" % base_image)
    with run_init_system_image(base_image, path=PROJECT_DIR, dockerfile=dockerfile) as [cont, backend]:
        try:
            # install latest agent
            run_chef_client(cont)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")
            ), "Datapoints didn't come through"
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))


@pytest.mark.upgrade_downgrade
@pytest.mark.parametrize("base_image,init_system", SUPPORTED_DISTROS)
def test_chef_upgrade_downgrade(base_image, init_system):
    dockerfile = os.path.join(DOCKERFILES_DIR, "Dockerfile.%s" % base_image)
    with run_init_system_image(base_image, path=PROJECT_DIR, dockerfile=dockerfile) as [cont, backend]:
        try:
            agent_version = run_chef_client(cont, "4.1.1")
            assert agent_version == "4.1.1", "agent version is not 4.1.1: %s" % agent_version
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")
            ), "Datapoints didn't come through"

            # upgrade agent
            agent_version = run_chef_client(cont, "4.2.0")
            assert agent_version == "4.2.0", "agent version is not 4.2.0: %s" % agent_version
            backend.reset_datapoints()
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")
            ), "Datapoints didn't come through"

            # downgrade agent for distros that support package downgrades
            if base_image not in ("debian-7-wheezy", "debian-8-jessie", "ubuntu1404"):
                agent_version = run_chef_client(cont, "4.1.0")
                assert agent_version == "4.1.0", "agent version is not 4.1.0: %s" % agent_version
                backend.reset_datapoints()
                assert wait_for(
                    p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")
                ), "Datapoints didn't come through"
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))

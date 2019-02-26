# Tests of the chef cookbook

import os
from functools import partial as p

import pytest

from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.util import print_lines, wait_for

from tests.packaging.common import (
    INIT_SYSTEMD,
    INIT_SYSV,
    INIT_UPSTART,
    PROJECT_DIR,
    get_agent_logs,
    is_agent_running_as_non_root,
    run_init_system_image,
)

pytestmark = pytest.mark.chef

CHEF_CMD = "chef-client -z -o 'recipe[signalfx_agent::default]' -j cookbooks/signalfx_agent/attributes.json"
DOCKERFILES_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), "images"))

SUPPORTED_DISTROS = [
    ("debian-7-wheezy", INIT_SYSV),
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


@pytest.mark.parametrize("base_image,init_system", SUPPORTED_DISTROS)
def test_chef(base_image, init_system):
    if base_image in ("debian-7-wheezy", "ubuntu1404"):
        pytest.skip("Wait for fix in debian init.d script to be released")
    dockerfile = os.path.join(DOCKERFILES_DIR, "Dockerfile.%s" % base_image)
    with run_init_system_image(base_image, path=PROJECT_DIR, dockerfile=dockerfile) as [cont, backend]:
        code, output = cont.exec_run(CHEF_CMD)
        print(output.decode("utf-8"))
        assert code == 0, "failed to install agent"

        try:
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")
            ), "Datapoints didn't come through"
            assert is_agent_running_as_non_root(cont), "Agent is not running as non-root user"
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))

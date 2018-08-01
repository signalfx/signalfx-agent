# Tests of the installer script

import docker
from functools import partial as p
import os
import pytest
import time

from .common import (
    INSTALLER_PATH,
    INIT_SYSV,
    INIT_UPSTART,
    INIT_SYSTEMD,
    build_base_image,
    get_agent_logs,
    get_rpm_package_to_test,
    get_deb_package_to_test,
    socat_https_proxy,
    run_init_system_image,
    copy_file_into_container,
)

from tests.helpers import fake_backend
from tests.helpers.assertions import *
from tests.helpers.util import run_container, wait_for, print_lines

pytestmark = pytest.mark.installer


def is_agent_running_as_non_root(container):
    code, output = container.exec_run("pgrep -u signalfx-agent signalfx-agent")
    print("pgrep check: %s" % output)
    return code == 0

supported_distros = [
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

@pytest.mark.parametrize("base_image,init_system", supported_distros)
def test_intaller(base_image, init_system):
    with run_init_system_image(base_image) as [cont, backend]:
        copy_file_into_container(INSTALLER_PATH, cont, "/opt/install.sh")

        # Unfortunately, wget and curl both don't like self-signed certs, even
        # if they are in the system bundle, so we need to use the --insecure
        # flag.
        code, output = cont.exec_run("sh /opt/install.sh MYTOKEN --beta --insecure")
        print("Output of install script:")
        print_lines(output)
        assert code == 0, "Agent could not be installed!"

        try:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")), "Datapoints didn't come through"
            assert is_agent_running_as_non_root(cont), "Agent is not running as non-root user"

            cont.stop(timeout=3)
            cont.start()

            backend.datapoints.clear()

            assert wait_for(p(is_agent_running_as_non_root, cont)), "Agent is not running as non-root user"
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")), "Datapoints didn't come through after restart"
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))


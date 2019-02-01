# Tests of the installer script

from functools import partial as p

import pytest

from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.util import print_lines, wait_for

from .common import (
    INIT_SYSTEMD,
    INIT_SYSV,
    INIT_UPSTART,
    INSTALLER_PATH,
    copy_file_into_container,
    get_agent_logs,
    is_agent_running_as_non_root,
    run_init_system_image,
)

pytestmark = pytest.mark.installer

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
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")
            ), "Datapoints didn't come through"
            assert is_agent_running_as_non_root(cont), "Agent is not running as non-root user"
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))

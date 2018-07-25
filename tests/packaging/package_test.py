from functools import partial as p
import os
import pytest
import time

from .common import (
    build_base_image,
    get_agent_logs,
    get_rpm_package_to_test,
    get_deb_package_to_test,
    socat_https_proxy,
    copy_file_into_container,
    run_init_system_image,
    INIT_SYSV,
    INIT_UPSTART,
    INIT_SYSTEMD,
    INSTALLER_PATH,
)

from tests.helpers import fake_backend
from tests.helpers.assertions import *
from tests.helpers.util import run_container, wait_for, print_lines

pytestmark = pytest.mark.packaging

PACKAGE_UTIL = {
    ".deb": "dpkg",
    ".rpm": "rpm",
}

INIT_START_COMMAND = {
    INIT_SYSV: "service signalfx-agent start",
    INIT_UPSTART: "/etc/init.d/signalfx-agent start",
    INIT_SYSTEMD: "systemctl start signalfx-agent",
}

INIT_LIST_COMMAND = {
    INIT_SYSV: "service --status-all",
    INIT_UPSTART: "chkconfig --list || service --status-all",
    INIT_SYSTEMD: "systemctl list-unit-files --type=service --no-pager",
}

INIT_STATUS_COMMAND = {
    INIT_SYSV: "service signalfx-agent status",
    INIT_UPSTART: "/etc/init.d/signalfx-agent status",
    INIT_SYSTEMD: "systemctl status signalfx-agent",
}


def is_agent_running_as_non_root(container):
    code, output = container.exec_run("pgrep -u signalfx-agent signalfx-agent")
    print("pgrep check: %s" % output)
    return code == 0


def _test_package_install(base_image, package_path, init_system):
    with run_init_system_image(base_image) as [cont, backend]:
        _, package_ext = os.path.splitext(package_path)
        copy_file_into_container(package_path, cont, "/opt/signalfx-agent%s" % package_ext)

        INSTALL_COMMAND = {
            ".rpm": "yum --nogpgcheck localinstall -y /opt/signalfx-agent.rpm",
            ".deb": "dpkg -i /opt/signalfx-agent.deb",
        }
        
        code, output = cont.exec_run(INSTALL_COMMAND[package_ext])
        print("Output of package install:")
        print_lines(output)
        assert code == 0, "Package could not be installed!"

        cont.exec_run("bash -ec 'echo -n testing > /etc/signalfx/token'")

        code, output = cont.exec_run(INIT_START_COMMAND[init_system])
        print("Init start command output:")
        print_lines(output)
        try:
            assert code == 0, "Agent could not be started"
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")), "Datapoints didn't come through"
            assert is_agent_running_as_non_root(cont)
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))


def _test_package_upgrade(base_image, package_path, init_system):
    with run_init_system_image(base_image) as [cont, backend]:
        _, package_ext = os.path.splitext(package_path)
        copy_file_into_container(package_path, cont, "/opt/signalfx-agent%s" % package_ext)
        copy_file_into_container(INSTALLER_PATH, cont, "/opt/install.sh")

        INSTALL_COMMAND = "sh /opt/install.sh testing123 --insecure --package-version 3.0.1-1"

        code, output = cont.exec_run(INSTALL_COMMAND)
        print("Output of old package install:")
        print_lines(output)
        assert code == 0, "Old package could not be installed!"

        UPGRADE_COMMAND = {
            ".rpm": "yum --nogpgcheck update -y /opt/signalfx-agent.rpm",
            ".deb": "dpkg -i /opt/signalfx-agent.deb",
        }
        
        code, output = cont.exec_run(UPGRADE_COMMAND[package_ext])
        print("Output of package upgrade:")
        print_lines(output)
        assert code == 0, "Package could not be upgraded!"

        code, output = cont.exec_run(["/bin/sh", "-c", INIT_STATUS_COMMAND[init_system]])
        print("Init status command output:")
        print_lines(output)
        assert code == 0, "Agent service not started after upgrade!"

        code, output = cont.exec_run(["/bin/sh", "-c", INIT_LIST_COMMAND[init_system]])
        print("Init list command output:")
        print_lines(output)
        output = output.decode('utf-8')
        assert code == 0, "Failed to get service list!"
        assert "signalfx-agent" in output, "Agent service not registered"

        try:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")), "Datapoints didn't come through"
            assert is_agent_running_as_non_root(cont)
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))


@pytest.mark.rpm
@pytest.mark.parametrize("base_image,init_system", [
    ("amazonlinux1", INIT_UPSTART),
    ("amazonlinux2", INIT_SYSTEMD),
    ("centos6", INIT_UPSTART),
    ("centos7", INIT_SYSTEMD),
])
def test_rpm_package(base_image, init_system):
    _test_package_install(base_image, get_rpm_package_to_test(), init_system)

@pytest.mark.deb
@pytest.mark.parametrize("base_image,init_system", [
    ("debian-7-wheezy", INIT_SYSV),
    ("debian-8-jessie", INIT_SYSTEMD),
    ("debian-9-stretch", INIT_SYSTEMD),
    ("ubuntu1404", INIT_UPSTART),
    ("ubuntu1604", INIT_SYSTEMD),
])
def test_deb_package(base_image, init_system):
    _test_package_install(base_image, get_deb_package_to_test(), init_system)

@pytest.mark.rpm
@pytest.mark.upgrade
@pytest.mark.parametrize("base_image,init_system", [
    ("amazonlinux1", INIT_UPSTART),
    ("amazonlinux2", INIT_SYSTEMD),
    ("centos6", INIT_UPSTART),
    ("centos7", INIT_SYSTEMD),
])
def test_rpm_package_upgrade(base_image, init_system):
    _test_package_upgrade(base_image, get_rpm_package_to_test(), init_system)

@pytest.mark.deb
@pytest.mark.upgrade
@pytest.mark.parametrize("base_image,init_system", [
    ("debian-7-wheezy", INIT_SYSV),
    ("debian-8-jessie", INIT_SYSTEMD),
    ("debian-9-stretch", INIT_SYSTEMD),
    ("ubuntu1404", INIT_UPSTART),
    ("ubuntu1604", INIT_SYSTEMD),
])
def test_deb_package_upgrade(base_image, init_system):
    _test_package_upgrade(base_image, get_deb_package_to_test(), init_system)

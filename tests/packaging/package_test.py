from functools import partial as p
import difflib
import docker
import os
import pytest
import re
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

AGENT_YAML_PATH = "/etc/signalfx/agent.yaml"
PIDFILE_PATH = "/var/run/signalfx-agent.pid"

INIT_START_TIMEOUT = 5
INIT_STOP_TIMEOUT = 11
INIT_RESTART_TIMEOUT = INIT_STOP_TIMEOUT + INIT_START_TIMEOUT

INIT_START_COMMAND = {
    INIT_SYSV: "service signalfx-agent start",
    INIT_UPSTART: "/etc/init.d/signalfx-agent start",
    INIT_SYSTEMD: "systemctl start signalfx-agent",
}

INIT_RESTART_COMMAND = {
    INIT_SYSV: "service signalfx-agent restart",
    INIT_UPSTART: "/etc/init.d/signalfx-agent restart",
    INIT_SYSTEMD: "systemctl restart signalfx-agent",
}

INIT_STOP_COMMAND = {
    INIT_SYSV: "service signalfx-agent stop",
    INIT_UPSTART: "/etc/init.d/signalfx-agent stop",
    INIT_SYSTEMD: "systemctl stop signalfx-agent",
}

INIT_LIST_COMMAND = {
    INIT_SYSV: "service --status-all",
    INIT_UPSTART: "bash -c 'chkconfig --list || service --status-all'",
    INIT_SYSTEMD: "systemctl list-unit-files --type=service --no-pager",
}

INIT_STATUS_COMMAND = {
    INIT_SYSV: "service signalfx-agent status",
    INIT_UPSTART: "/etc/init.d/signalfx-agent status",
    INIT_SYSTEMD: "systemctl status signalfx-agent",
}

INIT_STATUS_OUTPUT = {
    INIT_SYSV: {'active': "Running with pid", 'inactive': 'Not running'},
    INIT_UPSTART: {'active': "Running with pid", 'inactive': 'Not running'},
    INIT_SYSTEMD: {'active': 'Active: active (running)', 'inactive': 'Active: inactive (dead)'},
}


def is_agent_running_as_non_root(container):
    code, output = container.exec_run("pgrep -u signalfx-agent signalfx-agent")
    print("pgrep check: %s" % output)
    return code == 0


def get_agent_pid(container):
    command = "pgrep -u signalfx-agent -f /usr/bin/signalfx-agent"
    code, output = container.exec_run(command)
    output = output.decode('utf-8').strip()
    if code == 0:
        assert re.match('\d+', output), "Unexpected output from command '%s':\n%s" % (command, output)
        return output
    return None


def agent_has_new_pid(container, old_pid):
    def _new_pid():
        pid = get_agent_pid(container)
        return pid and pid != old_pid

    return wait_for(_new_pid, timeout_seconds=INIT_RESTART_TIMEOUT)


def path_exists_in_container(container, path):
    code, _ = container.exec_run("test -e %s" % path)
    return code == 0


def get_agent_yaml_diff(old_agent_yaml, new_agent_yaml):
    diff = "\n".join(
        difflib.unified_diff(
            old_agent_yaml.splitlines(),
            new_agent_yaml.splitlines(),
            fromfile="%s.orig" % AGENT_YAML_PATH,
            tofile=AGENT_YAML_PATH,
            lineterm='')).strip()
    return diff


def _test_service_status(container, init_system, expected_status):
    code, output = container.exec_run(INIT_STATUS_COMMAND[init_system])
    print("Init status command output:")
    print_lines(output)
    assert INIT_STATUS_OUTPUT[init_system][expected_status] in output.decode('utf-8'), \
        "'%s' expected in status output" % INIT_STATUS_OUTPUT[init_system][expected_status]


def _test_service_list(container, init_system, service_name="signalfx-agent"):
    code, output = container.exec_run(INIT_LIST_COMMAND[init_system])
    print("Init list command output:")
    print_lines(output)
    assert code == 0, "Failed to get service list"
    assert service_name in output.decode('utf-8'), "Agent service not registered"


def _test_service_start(container, init_system, backend):
    code, output = container.exec_run(INIT_START_COMMAND[init_system])
    print("Init start command output:")
    print_lines(output)
    backend.datapoints.clear()
    assert code == 0, "Agent could not be started"
    assert wait_for(p(is_agent_running_as_non_root, container), timeout_seconds=INIT_START_TIMEOUT)
    assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")), "Datapoints didn't come through"


def _test_service_restart(container, init_system, backend):
    old_pid = get_agent_pid(container)
    code, output = container.exec_run(INIT_RESTART_COMMAND[init_system])
    print("Init restart command output:")
    print_lines(output)
    backend.datapoints.clear()
    assert code == 0, "Agent could not be restarted"
    assert wait_for(p(is_agent_running_as_non_root, container), timeout_seconds=INIT_RESTART_TIMEOUT)
    assert agent_has_new_pid(container, old_pid), "Agent pid the same after service restart"
    assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")), "Datapoints didn't come through"


def _test_service_stop(container, init_system, backend):
    code, output = container.exec_run(INIT_STOP_COMMAND[init_system])
    print("Init stop command output:")
    print_lines(output)
    assert code == 0, "Agent could not be stop"
    assert wait_for(lambda: not get_agent_pid(container), timeout_seconds=INIT_STOP_TIMEOUT), "Timed out waiting for agent process to stop"
    if init_system in [INIT_SYSV, INIT_UPSTART]:
        assert not path_exists_in_container(container, PIDFILE_PATH), "%s exists when agent is stopped" % PIDFILE_PATH
    backend.datapoints.clear()


def _test_system_restart(container, init_system, backend):
    print("Restarting container")
    try:
        container.stop(timeout=3)
        backend.datapoints.clear()
        container.start()
        assert wait_for(p(is_agent_running_as_non_root, container), timeout_seconds=INIT_RESTART_TIMEOUT)
        _test_service_status(container, init_system, 'active')
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")), "Datapoints didn't come through"
    except docker.errors.APIError as e:
        if not "id already in use" in str(e).lower():
            raise e
        else:
            # possible intermittent bug with docker daemon not restarting containers correctly
            print("Container failed to restart:\n%s" % str(e))


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

        try:
            _test_service_list(cont, init_system)
            _test_service_start(cont, init_system, backend)
            _test_service_status(cont, init_system, 'active')
            _test_service_restart(cont, init_system, backend)
            _test_service_status(cont, init_system, 'active')
            _test_service_stop(cont, init_system, backend)
            _test_service_status(cont, init_system, 'inactive')
            _test_system_restart(cont, init_system, backend)
        finally:
            cont.reload()
            if cont.status.lower() == 'running':
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

        assert path_exists_in_container(cont, AGENT_YAML_PATH), "%s does not exist!" % AGENT_YAML_PATH

        _, _ = cont.exec_run("bash -ec 'echo >> %s'" % AGENT_YAML_PATH)
        _, _ = cont.exec_run("bash -ec 'echo \"hostname: test-host\" >> %s'" % AGENT_YAML_PATH)
        _, _ = cont.exec_run("cp -f %s %s.orig" % (AGENT_YAML_PATH, AGENT_YAML_PATH))
        _, output = cont.exec_run("cat %s" % AGENT_YAML_PATH)
        old_agent_yaml = output.decode('utf-8')

        UPGRADE_COMMAND = {
            ".rpm": "yum --nogpgcheck update -y /opt/signalfx-agent.rpm",
            ".deb": "dpkg -i --force-confold /opt/signalfx-agent.deb",
        }

        code, output = cont.exec_run(UPGRADE_COMMAND[package_ext])
        print("Output of package upgrade:")
        print_lines(output)
        assert code == 0, "Package could not be upgraded!"

        assert path_exists_in_container(cont, AGENT_YAML_PATH), "%s does not exist after upgrade!" % AGENT_YAML_PATH

        new_agent_yaml = cont.exec_run("cat %s" % AGENT_YAML_PATH)[1].decode('utf-8')
        diff = get_agent_yaml_diff(old_agent_yaml, new_agent_yaml)
        assert len(diff) == 0, "%s different after upgrade!\n%s" % (AGENT_YAML_PATH, diff)

        try:
            _test_service_list(cont, init_system)
            _test_service_status(cont, init_system, 'active')
            _test_service_restart(cont, init_system, backend)
            _test_service_status(cont, init_system, 'active')
            _test_service_stop(cont, init_system, backend)
            _test_service_status(cont, init_system, 'inactive')
            _test_service_start(cont, init_system, backend)
            _test_service_status(cont, init_system, 'active')
            _test_service_stop(cont, init_system, backend)
            _test_system_restart(cont, init_system, backend)
        finally:
            cont.reload()
            if cont.status.lower() == 'running':
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
    ("ubuntu1804", INIT_SYSTEMD),
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
    ("ubuntu1804", INIT_SYSTEMD),
])
def test_deb_package_upgrade(base_image, init_system):
    _test_package_upgrade(base_image, get_deb_package_to_test(), init_system)

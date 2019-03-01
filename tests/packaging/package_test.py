import difflib
import os
import re
from functools import partial as p

import pytest
from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.util import print_lines, wait_for

from .common import (
    AGENT_YAML_PATH,
    INIT_SYSTEMD,
    INIT_SYSV,
    INIT_UPSTART,
    PIDFILE_PATH,
    copy_file_into_container,
    get_agent_logs,
    get_container_file_content,
    get_deb_package_to_test,
    get_rpm_package_to_test,
    is_agent_running_as_non_root,
    path_exists_in_container,
    run_init_system_image,
)

pytestmark = pytest.mark.packaging

PACKAGE_UTIL = {".deb": "dpkg", ".rpm": "rpm"}

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
    INIT_SYSV: {"active": "Running with pid", "inactive": "Not running"},
    INIT_UPSTART: {"active": "Running with pid", "inactive": "Not running"},
    INIT_SYSTEMD: {"active": "Active: active (running)", "inactive": "Active: inactive (dead)"},
}


def get_agent_pid(container):
    command = "pgrep -u signalfx-agent -f /usr/bin/signalfx-agent"
    code, output = container.exec_run(command)
    output = output.decode("utf-8").strip()
    if code == 0:
        assert re.match(r"\d+", output), "Unexpected output from command '%s':\n%s" % (command, output)
        return output
    return None


def agent_has_new_pid(container, old_pid):
    def _new_pid():
        pid = get_agent_pid(container)
        return pid and pid != old_pid

    return wait_for(_new_pid, timeout_seconds=INIT_RESTART_TIMEOUT)


def get_agent_yaml_diff(old_agent_yaml, new_agent_yaml):
    diff = "\n".join(
        difflib.unified_diff(
            old_agent_yaml.splitlines(),
            new_agent_yaml.splitlines(),
            fromfile="%s.orig" % AGENT_YAML_PATH,
            tofile=AGENT_YAML_PATH,
            lineterm="",
        )
    ).strip()
    return diff


def update_agent_yaml(container, backend, hostname="test-hostname"):
    def set_option(name, value):
        code, _ = container.exec_run("grep '^%s:' %s" % (name, AGENT_YAML_PATH))
        if code == 0:
            _, _ = container.exec_run("sed -i 's|^%s:.*|%s: %s|' %s" % (name, name, value, AGENT_YAML_PATH))
        else:
            _, _ = container.exec_run("bash -ec 'echo >> %s'" % AGENT_YAML_PATH)
            _, _ = container.exec_run("bash -ec 'echo \"%s: %s\" >> %s'" % (name, value, AGENT_YAML_PATH))

    assert path_exists_in_container(container, AGENT_YAML_PATH), "File %s does not exist!" % AGENT_YAML_PATH
    if hostname:
        set_option("hostname", hostname)
    ingest_url = "http://%s:%d" % (backend.ingest_host, backend.ingest_port)
    set_option("ingestUrl", ingest_url)
    api_url = "http://%s:%d" % (backend.api_host, backend.api_port)
    set_option("apiUrl", api_url)
    return get_container_file_content(container, AGENT_YAML_PATH)


def _test_service_status(container, init_system, expected_status):
    _, output = container.exec_run(INIT_STATUS_COMMAND[init_system])
    print("Init status command output:")
    print_lines(output)
    assert INIT_STATUS_OUTPUT[init_system][expected_status] in output.decode("utf-8"), (
        "'%s' expected in status output" % INIT_STATUS_OUTPUT[init_system][expected_status]
    )


def _test_service_list(container, init_system, service_name="signalfx-agent"):
    code, output = container.exec_run(INIT_LIST_COMMAND[init_system])
    print("Init list command output:")
    print_lines(output)
    assert code == 0, "Failed to get service list"
    assert service_name in output.decode("utf-8"), "Agent service not registered"


def _test_service_start(container, init_system, backend):
    code, output = container.exec_run(INIT_START_COMMAND[init_system])
    print("Init start command output:")
    print_lines(output)
    backend.reset_datapoints()
    assert code == 0, "Agent could not be started"
    assert wait_for(p(is_agent_running_as_non_root, container), timeout_seconds=INIT_START_TIMEOUT)
    assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")), "Datapoints didn't come through"


def _test_service_restart(container, init_system, backend):
    old_pid = get_agent_pid(container)
    code, output = container.exec_run(INIT_RESTART_COMMAND[init_system])
    print("Init restart command output:")
    print_lines(output)
    backend.reset_datapoints()
    assert code == 0, "Agent could not be restarted"
    assert wait_for(p(is_agent_running_as_non_root, container), timeout_seconds=INIT_RESTART_TIMEOUT)
    assert agent_has_new_pid(container, old_pid), "Agent pid the same after service restart"
    assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")), "Datapoints didn't come through"


def _test_service_stop(container, init_system, backend):
    code, output = container.exec_run(INIT_STOP_COMMAND[init_system])
    print("Init stop command output:")
    print_lines(output)
    assert code == 0, "Agent could not be stop"
    assert wait_for(
        lambda: not get_agent_pid(container), timeout_seconds=INIT_STOP_TIMEOUT
    ), "Timed out waiting for agent process to stop"
    if init_system in [INIT_SYSV, INIT_UPSTART]:
        assert not path_exists_in_container(container, PIDFILE_PATH), "%s exists when agent is stopped" % PIDFILE_PATH
    backend.reset_datapoints()


def _test_system_restart(container, init_system, backend):
    print("Restarting container")
    container.stop(timeout=3)
    backend.reset_datapoints()
    container.start()
    assert wait_for(p(is_agent_running_as_non_root, container), timeout_seconds=INIT_RESTART_TIMEOUT)
    _test_service_status(container, init_system, "active")
    assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")), "Datapoints didn't come through"


INSTALL_COMMAND = {
    ".rpm": "yum --nogpgcheck localinstall -y /opt/signalfx-agent.rpm",
    ".deb": "dpkg -i /opt/signalfx-agent.deb",
}


def _test_package_install(base_image, package_path, init_system):
    with run_init_system_image(base_image, with_socat=False) as [cont, backend]:
        _, package_ext = os.path.splitext(package_path)
        copy_file_into_container(package_path, cont, "/opt/signalfx-agent%s" % package_ext)

        code, output = cont.exec_run(INSTALL_COMMAND[package_ext])
        print("Output of package install:")
        print_lines(output)
        assert code == 0, "Package could not be installed!"

        cont.exec_run("bash -ec 'echo -n testing123 > /etc/signalfx/token'")
        update_agent_yaml(cont, backend, hostname="test-" + base_image)

        try:
            _test_service_list(cont, init_system)
            _test_service_restart(cont, init_system, backend)
            _test_service_status(cont, init_system, "active")
            _test_service_stop(cont, init_system, backend)
            _test_service_status(cont, init_system, "inactive")
            _test_service_start(cont, init_system, backend)
            _test_service_status(cont, init_system, "active")
            _test_service_stop(cont, init_system, backend)
            _test_system_restart(cont, init_system, backend)
        finally:
            cont.reload()
            if cont.status.lower() == "running":
                print("Agent log:")
                print_lines(get_agent_logs(cont, init_system))


# pylint: disable=line-too-long
OLD_INSTALL_COMMAND = {
    ".rpm": "yum install -y https://s3.amazonaws.com/public-downloads--signalfuse-com/rpms/signalfx-agent/final/signalfx-agent-3.0.1-1.x86_64.rpm",
    ".deb": "bash -ec 'wget -nv https://s3.amazonaws.com/public-downloads--signalfuse-com/debs/signalfx-agent/final/pool/signalfx-agent_3.0.1-1_amd64.deb && dpkg -i signalfx-agent_3.0.1-1_amd64.deb'",
}

UPGRADE_COMMAND = {
    ".rpm": "yum --nogpgcheck update -y /opt/signalfx-agent.rpm",
    ".deb": "dpkg -i --force-confold /opt/signalfx-agent.deb",
}


def _test_package_upgrade(base_image, package_path, init_system):
    with run_init_system_image(base_image, with_socat=False) as [cont, backend]:
        _, package_ext = os.path.splitext(package_path)
        copy_file_into_container(package_path, cont, "/opt/signalfx-agent%s" % package_ext)

        code, output = cont.exec_run(OLD_INSTALL_COMMAND[package_ext])
        print("Output of old package install:")
        print_lines(output)
        assert code == 0, "Old package could not be installed!"

        cont.exec_run("bash -ec 'echo -n testing123 > /etc/signalfx/token'")
        old_agent_yaml = update_agent_yaml(cont, backend, hostname="test-" + base_image)
        _, _ = cont.exec_run("cp -f %s %s.orig" % (AGENT_YAML_PATH, AGENT_YAML_PATH))

        code, output = cont.exec_run(UPGRADE_COMMAND[package_ext])
        print("Output of package upgrade:")
        print_lines(output)
        assert code == 0, "Package could not be upgraded!"

        new_agent_yaml = get_container_file_content(cont, AGENT_YAML_PATH)
        diff = get_agent_yaml_diff(old_agent_yaml, new_agent_yaml)
        assert not diff, "%s different after upgrade!\n%s" % (AGENT_YAML_PATH, diff)

        try:
            _test_service_list(cont, init_system)
            _test_service_restart(cont, init_system, backend)
            _test_service_status(cont, init_system, "active")
            _test_service_stop(cont, init_system, backend)
            _test_service_status(cont, init_system, "inactive")
            _test_service_start(cont, init_system, backend)
            _test_service_status(cont, init_system, "active")
            _test_service_stop(cont, init_system, backend)
            _test_system_restart(cont, init_system, backend)
        finally:
            cont.reload()
            if cont.status.lower() == "running":
                print("Agent log:")
                print_lines(get_agent_logs(cont, init_system))


@pytest.mark.rpm
@pytest.mark.parametrize(
    "base_image,init_system",
    [
        ("amazonlinux1", INIT_UPSTART),
        ("amazonlinux2", INIT_SYSTEMD),
        ("centos6", INIT_UPSTART),
        ("centos7", INIT_SYSTEMD),
    ],
)
def test_rpm_package(base_image, init_system):
    _test_package_install(base_image, get_rpm_package_to_test(), init_system)


@pytest.mark.deb
@pytest.mark.parametrize(
    "base_image,init_system",
    [
        ("debian-7-wheezy", INIT_SYSV),
        ("debian-8-jessie", INIT_SYSTEMD),
        ("debian-9-stretch", INIT_SYSTEMD),
        ("ubuntu1404", INIT_UPSTART),
        ("ubuntu1604", INIT_SYSTEMD),
        ("ubuntu1804", INIT_SYSTEMD),
    ],
)
def test_deb_package(base_image, init_system):
    _test_package_install(base_image, get_deb_package_to_test(), init_system)


@pytest.mark.rpm
@pytest.mark.upgrade
@pytest.mark.parametrize(
    "base_image,init_system",
    [
        ("amazonlinux1", INIT_UPSTART),
        ("amazonlinux2", INIT_SYSTEMD),
        ("centos6", INIT_UPSTART),
        ("centos7", INIT_SYSTEMD),
    ],
)
def test_rpm_package_upgrade(base_image, init_system):
    _test_package_upgrade(base_image, get_rpm_package_to_test(), init_system)


@pytest.mark.deb
@pytest.mark.upgrade
@pytest.mark.parametrize(
    "base_image,init_system",
    [
        ("debian-7-wheezy", INIT_SYSV),
        ("debian-8-jessie", INIT_SYSTEMD),
        ("debian-9-stretch", INIT_SYSTEMD),
        ("ubuntu1404", INIT_UPSTART),
        ("ubuntu1604", INIT_SYSTEMD),
        ("ubuntu1804", INIT_SYSTEMD),
    ],
)
def test_deb_package_upgrade(base_image, init_system):
    _test_package_upgrade(base_image, get_deb_package_to_test(), init_system)

import difflib
import os
import re
from functools import partial as p

import pytest
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import (
    copy_file_into_container,
    get_container_file_content,
    path_exists_in_container,
    print_lines,
    wait_for,
)

from tests.packaging.common import (
    AGENT_YAML_PATH,
    INIT_SYSTEMD,
    INIT_SYSV,
    INIT_UPSTART,
    PIDFILE_PATH,
    get_agent_logs,
    get_deb_package_to_test,
    get_rpm_package_to_test,
    is_agent_running_as_non_root,
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


def install_deps(cont, base_image):
    if "debian" in base_image or "ubuntu" in base_image:
        cmd = "sh -ec 'apt-get update && apt-get install -y libcap2-bin'"
    elif "opensuse" in base_image:
        cmd = "zypper install -y -l libcap2 libcap-progs libpcap1 shadow"
    else:
        return

    code, output = cont.exec_run(cmd)
    assert code == 0, "Failed to install dependencies:\n%s" % output.decode("utf-8")


def _test_install_package(cont, cmd):
    code, output = cont.exec_run(cmd)
    print("Output of package install:")
    print_lines(output)
    assert code == 0, "Package could not be installed!"


def _test_service_status(container, init_system, expected_status):
    _, output = container.exec_run(INIT_STATUS_COMMAND[init_system])
    print("%s status command output:" % init_system)
    print_lines(output)
    assert INIT_STATUS_OUTPUT[init_system][expected_status] in output.decode("utf-8"), (
        "'%s' expected in status output" % INIT_STATUS_OUTPUT[init_system][expected_status]
    )


def _test_service_list(container, init_system, service_name="signalfx-agent"):
    code, output = container.exec_run(INIT_LIST_COMMAND[init_system])
    print("%s list command output:" % init_system)
    print_lines(output)
    assert code == 0, "Failed to get service list"
    assert service_name in output.decode("utf-8"), "Agent service not registered"


def _test_service_start(container, init_system, backend, user="signalfx-agent"):
    code, output = container.exec_run(INIT_START_COMMAND[init_system])
    print("%s start command output:" % init_system)
    print_lines(output)
    backend.reset_datapoints()
    assert code == 0, "Agent could not be started"
    assert wait_for(p(is_agent_running_as_non_root, container, user=user), timeout_seconds=INIT_START_TIMEOUT)
    assert wait_for(p(has_datapoint, backend, metric_name="disk.utilization")), "Datapoints didn't come through"


def _test_service_restart(container, init_system, backend):
    old_pid = get_agent_pid(container)
    code, output = container.exec_run(INIT_RESTART_COMMAND[init_system])
    print("%s restart command output:" % init_system)
    print_lines(output)
    backend.reset_datapoints()
    assert code == 0, "Agent could not be restarted"
    assert wait_for(p(is_agent_running_as_non_root, container), timeout_seconds=INIT_RESTART_TIMEOUT)
    assert agent_has_new_pid(container, old_pid), "Agent pid the same after service restart"
    assert wait_for(p(has_datapoint, backend, metric_name="disk.utilization")), "Datapoints didn't come through"


def _test_service_stop(container, init_system, backend):
    code, output = container.exec_run(INIT_STOP_COMMAND[init_system])
    print("%s stop command output:" % init_system)
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
    assert wait_for(p(has_datapoint, backend, metric_name="disk.utilization")), "Datapoints didn't come through"


def _test_service_status_redirect(container):
    code, output = container.exec_run(INIT_STATUS_COMMAND[INIT_SYSV])
    print("%s status command output:" % INIT_SYSV)
    print_lines(output)
    assert code == 0 and "/lib/systemd/system/signalfx-agent.service; enabled" in output.decode("utf-8")


def _test_agent_status(container):
    for arg in ["", "all", "monitors", "config", "endpoints"]:
        cmd = f"agent-status {arg}" if arg else "agent-status"
        print(f"Running '{cmd}' ...")
        code, output = container.exec_run(cmd)
        assert code == 0, output.decode("utf-8")


def _test_package_verify(container, package_ext, base_image):
    if package_ext == ".rpm":
        if "opensuse" in base_image:
            code, output = container.exec_run("rpm --verify --nodeps signalfx-agent")
        else:
            code, output = container.exec_run("rpm --verify signalfx-agent")
        assert code == 0, "rpm verify failed!\n%s" % output.decode("utf-8")
    elif package_ext == ".deb":
        code, output = container.exec_run("dpkg --verify signalfx-agent")
        assert code == 0, "dpkg verify failed!\n%s" % output.decode("utf-8")


def _test_service_override(container, init_system, backend):
    _test_service_stop(container, init_system, backend)

    code, output = container.exec_run(f"useradd --system --user-group --no-create-home --shell /sbin/nologin test-user")
    assert code == 0, output.decode("utf-8")

    if init_system == INIT_SYSTEMD:
        override_path = "/etc/systemd/system/signalfx-agent.service.d/override.conf"
        config = "[Service]\nUser=test-user\nGroup=test-user"
    else:
        override_path = "/etc/default/signalfx-agent"
        config = "user=test-user\ngroup=test-user"

    container.exec_run(f"mkdir -p {os.path.dirname(override_path)}")
    container.exec_run(f'''bash -c "echo -e '{config}' > {override_path}"''')

    if init_system == INIT_SYSTEMD:
        # override tmpfile with the new user/group
        tmpfile_override_path = "/etc/tmpfiles.d/signalfx-agent.conf"
        tmpfile_override_config = "D /run/signalfx-agent 0755 test-user test-user - -"
        container.exec_run(f'''bash -c "echo '{tmpfile_override_config}' > {tmpfile_override_path}"''')
        code, output = container.exec_run(f"systemd-tmpfiles --create --remove {tmpfile_override_path}")
        assert code == 0, output.decode("utf-8")

        code, output = container.exec_run("systemctl daemon-reload")
        assert code == 0, output.decode("utf-8")

    _test_service_start(container, init_system, backend, user="test-user")

    _test_service_status(container, init_system, "active")


INSTALL_COMMAND = {
    ".rpm": "yum --nogpgcheck localinstall -y /opt/signalfx-agent.rpm",
    ".deb": "dpkg -i /opt/signalfx-agent.deb",
}


def _test_package_install(base_image, package_path, init_system):
    with run_init_system_image(base_image, with_socat=False) as [cont, backend]:
        _, package_ext = os.path.splitext(package_path)
        copy_file_into_container(package_path, cont, "/opt/signalfx-agent%s" % package_ext)

        install_deps(cont, base_image)

        if "opensuse" in base_image:
            install_cmd = "rpm -ivh --nodeps /opt/signalfx-agent.rpm"
        else:
            install_cmd = INSTALL_COMMAND[package_ext]

        _test_install_package(cont, install_cmd)

        _test_package_verify(cont, package_ext, base_image)

        if init_system == INIT_SYSTEMD:
            assert not path_exists_in_container(cont, "/etc/init.d/signalfx-agent")
        else:
            assert path_exists_in_container(cont, "/etc/init.d/signalfx-agent")

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
            if init_system == INIT_SYSTEMD:
                _test_service_status_redirect(cont)
            _test_agent_status(cont)
            _test_service_override(cont, init_system, backend)
        finally:
            cont.reload()
            if cont.status.lower() == "running":
                print("Agent service status:")
                print_lines(cont.exec_run(INIT_STATUS_COMMAND[init_system]).output)
                print("Agent log:")
                print_lines(get_agent_logs(cont, init_system))


OLD_RPM_URL = "https://dl.signalfx.com/rpms/signalfx-agent/archive/release/signalfx-agent-3.0.1-1.x86_64.rpm"
OLD_RPM_NAME = OLD_RPM_URL.split("/")[-1]
OLD_SUSE_RPM_URL = "https://splunk.jfrog.io/splunk/signalfx-agent-rpm/release/signalfx-agent-4.7.7-1.x86_64.rpm"
OLD_SUSE_RPM_NAME = OLD_SUSE_RPM_URL.split("/")[-1]
OLD_DEB_URL = "https://dl.signalfx.com/debs/signalfx-agent/archive/release/signalfx-agent_3.0.1-1_amd64.deb"
OLD_DEB_NAME = OLD_DEB_URL.split("/")[-1]

OLD_INSTALL_COMMAND = {
    ".rpm": f"yum install -y {OLD_RPM_URL}",
    ".deb": f"bash -ec 'wget -nv {OLD_DEB_URL} && dpkg -i {OLD_DEB_NAME}'",
}

UPGRADE_COMMAND = {
    ".rpm": "yum --nogpgcheck update -y /opt/signalfx-agent.rpm",
    ".deb": "dpkg -i --force-confold /opt/signalfx-agent.deb",
}


def _test_package_upgrade(base_image, package_path, init_system):
    with run_init_system_image(base_image, with_socat=False) as [cont, backend]:
        _, package_ext = os.path.splitext(package_path)
        copy_file_into_container(package_path, cont, "/opt/signalfx-agent%s" % package_ext)

        install_deps(cont, base_image)

        if "opensuse" in base_image:
            install_cmd = f"bash -ec 'wget -nv {OLD_SUSE_RPM_URL} && rpm -ivh --nodeps {OLD_SUSE_RPM_NAME}'"
            upgrade_cmd = "rpm -Uvh --nodeps /opt/signalfx-agent.rpm"
        else:
            install_cmd = OLD_INSTALL_COMMAND[package_ext]
            upgrade_cmd = UPGRADE_COMMAND[package_ext]

        _test_install_package(cont, install_cmd)

        cont.exec_run("bash -ec 'echo -n testing123 > /etc/signalfx/token'")
        old_agent_yaml = update_agent_yaml(cont, backend, hostname="test-" + base_image)
        _, _ = cont.exec_run("cp -f %s %s.orig" % (AGENT_YAML_PATH, AGENT_YAML_PATH))

        code, output = cont.exec_run(upgrade_cmd)
        print("Output of package upgrade:")
        print_lines(output)
        assert code == 0, "Package could not be upgraded!"

        _test_package_verify(cont, package_ext, base_image)

        if init_system == INIT_SYSTEMD:
            assert not path_exists_in_container(cont, "/etc/init.d/signalfx-agent")
        else:
            assert path_exists_in_container(cont, "/etc/init.d/signalfx-agent")

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
            if init_system == INIT_SYSTEMD:
                _test_service_status_redirect(cont)
            _test_agent_status(cont)
            _test_service_override(cont, init_system, backend)
        finally:
            cont.reload()
            if cont.status.lower() == "running":
                print("Agent service status:")
                print_lines(cont.exec_run(INIT_STATUS_COMMAND[init_system]).output)
                print("Agent log:")
                print_lines(get_agent_logs(cont, init_system))


@pytest.mark.rpm
@pytest.mark.parametrize(
    "base_image,init_system",
    [
        ("amazonlinux1", INIT_UPSTART),
        ("amazonlinux2", INIT_SYSTEMD),
        ("centos7", INIT_SYSTEMD),
        ("centos8", INIT_SYSTEMD),
        ("opensuse15", INIT_SYSTEMD),
    ],
)
def test_rpm_package(base_image, init_system):
    _test_package_install(base_image, get_rpm_package_to_test(), init_system)


@pytest.mark.deb
@pytest.mark.parametrize(
    "base_image,init_system",
    [
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
        ("centos7", INIT_SYSTEMD),
        ("centos8", INIT_SYSTEMD),
        ("opensuse15", INIT_SYSTEMD),
    ],
)
def test_rpm_package_upgrade(base_image, init_system):
    _test_package_upgrade(base_image, get_rpm_package_to_test(), init_system)


@pytest.mark.deb
@pytest.mark.upgrade
@pytest.mark.parametrize(
    "base_image,init_system",
    [
        ("debian-8-jessie", INIT_SYSTEMD),
        ("debian-9-stretch", INIT_SYSTEMD),
        ("ubuntu1404", INIT_UPSTART),
        ("ubuntu1604", INIT_SYSTEMD),
        ("ubuntu1804", INIT_SYSTEMD),
    ],
)
def test_deb_package_upgrade(base_image, init_system):
    _test_package_upgrade(base_image, get_deb_package_to_test(), init_system)

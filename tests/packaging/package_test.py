from functools import partial as p
import os
import pytest
import time

from .common import (
    build_base_image,
    get_rpm_package_to_test,
    get_deb_package_to_test,
    socat_https_proxy,
    copy_file_into_container,
)

from tests.helpers import fake_backend
from tests.helpers.assertions import *
from tests.helpers.util import run_container, wait_for, print_lines

pytestmark = pytest.mark.packaging

PACKAGE_UTIL = {
    ".deb": "dpkg",
    ".rpm": "rpm",
}

INIT_SYSV = "sysv"
INIT_UPSTART = "upstart"
INIT_SYSTEMD = "systemd"

INIT_START_COMMAND = {
    INIT_SYSV: "service signalfx-agent start",
    INIT_UPSTART: "bash -ec 'initctl reload-configuration && start signalfx-agent'",
    INIT_SYSTEMD: "systemctl start signalfx-agent",
}


def get_agent_logs(container, init_system):
    LOG_COMMAND = {
        INIT_SYSV: "cat /var/log/signalfx-agent.log",
        INIT_UPSTART: "cat /var/log/signalfx-agent.log",
        INIT_SYSTEMD: "journalctl -u signalfx-agent",
    }
    _, output = container.exec_run(LOG_COMMAND[init_system])
    return output


def is_agent_running_as_non_root(container):
    code, output = container.exec_run("pgrep -u signalfx-agent signalfx-agent")
    print("pgrep check: %s" % output)
    return code == 0


def _test_package_install(base_image, package_path, init_system):
    image_id = build_base_image(base_image)
    print("Image ID: %s" % image_id)
    with fake_backend.start() as backend:
        container_options = {
            # Init systems running in the container want permissions
            "privileged": True,
            "volumes": {
                "/sys/fs/cgroup": {"bind": "/sys/fs/cgroup", "mode": "ro"},
                "/tmp/scratch": {"bind": "/tmp/scratch", "mode": "rw"},
            },
            "extra_hosts": {
                # Socat will be running on localhost to forward requests to
                # these hosts to the fake backend
                "ingest.signalfx.com": '127.0.0.1',
                "api.signalfx.com": '127.0.0.1',
            },
        }
        with run_container(image_id, wait_for_ip=False, **container_options) as cont:
            # Proxy the backend calls through a fake HTTPS endpoint so that we
            # don't have to change the default configuration included by the
            # package.  The base_image used should trust the self-signed certs
            # included in the images dir so that the agent doesn't throw TLS
            # verification errors.
            with socat_https_proxy(cont, backend.ingest_host, backend.ingest_port, "ingest.signalfx.com"), \
                 socat_https_proxy(cont, backend.api_host, backend.api_port, "api.signalfx.com"):

                _, package_ext = os.path.splitext(package_path)
                copy_file_into_container(package_path, cont, "/tmp/signalfx-agent%s" % package_ext)

                INSTALL_COMMAND = {
                    ".rpm": "yum --nogpgcheck localinstall -y /tmp/signalfx-agent.rpm",
                    ".deb": "dpkg -i /tmp/signalfx-agent.deb",
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
                    
                    wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata"))
                    assert is_agent_running_as_non_root(cont)
                finally:
                    print("Agent log:")
                    print_lines(get_agent_logs(cont, init_system))


def _test_rpm_package(base_image, init_system):
    _test_package_install(base_image, get_rpm_package_to_test(), init_system)

def _test_deb_package(base_image, init_system):
    _test_package_install(base_image, get_deb_package_to_test(), init_system)


@pytest.mark.rpm
def test_amazon_linux_1_package():
    _test_rpm_package("amazonlinux1", INIT_UPSTART)

@pytest.mark.rpm
def test_amazon_linux_2_package():
    _test_rpm_package("amazonlinux2", INIT_SYSTEMD)

@pytest.mark.rpm
def test_centos6_package():
    _test_rpm_package("centos6", INIT_UPSTART)

@pytest.mark.rpm
def test_centos7_package():
    _test_rpm_package("centos7", INIT_SYSTEMD)

@pytest.mark.deb
def test_ubuntu1404_package():
    _test_deb_package("ubuntu1404", INIT_UPSTART)

@pytest.mark.deb
def test_ubuntu1604_package():
    _test_deb_package("ubuntu1604", INIT_SYSTEMD)

@pytest.mark.deb
def test_debian7_package():
    _test_deb_package("debian-7-wheezy", INIT_SYSV)

@pytest.mark.deb
def test_debian8_package():
    _test_deb_package("debian-8-jessie", INIT_SYSTEMD)

@pytest.mark.deb
def test_debian9_package():
    _test_deb_package("debian-9-stretch", INIT_SYSTEMD)

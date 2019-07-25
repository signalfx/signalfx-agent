# Tests of the installer script

from contextlib import contextmanager
from functools import partial as p

import pytest
from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.util import copy_file_into_container, print_lines, wait_for, wait_for_assertion

from .common import (
    INIT_SYSTEMD,
    INIT_UPSTART,
    INSTALLER_PATH,
    get_agent_logs,
    get_agent_version,
    is_agent_running_as_non_root,
    run_init_system_image,
)

pytestmark = pytest.mark.installer

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


@contextmanager
def _run_tests(base_image, init_system, installer_args, **extra_run_kwargs):
    with run_init_system_image(base_image, **extra_run_kwargs) as [cont, backend]:
        copy_file_into_container(INSTALLER_PATH, cont, "/opt/install.sh")

        # Unfortunately, wget and curl both don't like self-signed certs, even
        # if they are in the system bundle, so we need to use the --insecure
        # flag.
        code, output = cont.exec_run(f"sh /opt/install.sh --insecure {installer_args}")
        print("Output of install script:")
        print_lines(output)
        assert code == 0, "Agent could not be installed!"

        try:
            assert is_agent_running_as_non_root(cont), "Agent is running as root user"
            yield backend, cont
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))


@pytest.mark.parametrize("base_image,init_system", SUPPORTED_DISTROS)
def test_installer_on_all_distros(base_image, init_system):
    args = "MYTOKEN"
    with _run_tests(base_image, init_system, args) as [backend, _]:
        assert wait_for(
            p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")
        ), "Datapoints didn't come through"


@pytest.mark.parametrize("base_image,init_system", SUPPORTED_DISTROS)
@pytest.mark.parametrize("agent_version", ["4.7.8"])
def test_installer_package_version(base_image, init_system, agent_version):
    args = f"--package-version {agent_version}-1 MYTOKEN"
    with _run_tests(base_image, init_system, args) as [backend, cont]:
        installed_version = get_agent_version(cont)
        agent_version = agent_version.replace("~", "-")
        assert (
            installed_version == agent_version
        ), f"Installed agent version is {installed_version} but should be {agent_version}"
        assert wait_for(
            p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")
        ), "Datapoints didn't come through"


def test_installer_different_realm():
    with _run_tests(
        "ubuntu1804",
        INIT_SYSTEMD,
        "MYTOKEN --realm us1",
        ingest_host="ingest.us1.signalfx.com",
        api_host="api.us1.signalfx.com",
    ) as [backend, _]:
        assert wait_for(
            p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")
        ), "Datapoints didn't come through"


def first_host_dimension(backend):
    """
    Find the first value of the host dimension that comes through to the
    backend.
    """
    for dp in backend.datapoints:
        for dim in dp.dimensions:
            if dim.key == "host":
                return dim.value
    return None


@pytest.mark.xfail(reason="won't pass until agent is released with default config with cluster option referencing file")
def test_installer_cluster():
    with _run_tests("ubuntu1804", INIT_SYSTEMD, "MYTOKEN --cluster prod") as [backend, _]:

        def assert_cluster_property():
            host = first_host_dimension(backend)
            assert host
            assert host in backend.dims["host"]
            dim = backend.dims["host"][host]
            assert dim["customProperties"] == {"cluster": "prod"}
            assert dim["tags"] in [None, []]

        wait_for_assertion(assert_cluster_property)

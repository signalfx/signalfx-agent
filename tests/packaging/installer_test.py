# Tests of the installer script

import os
import sys
from contextlib import contextmanager
from functools import partial as p

import pytest

from tests.helpers.agent import ensure_fake_backend
from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.formatting import print_dp_or_event
from tests.helpers.util import copy_file_into_container, print_lines, wait_for, wait_for_assertion

from tests.packaging.common import (
    INIT_SYSTEMD,
    INIT_UPSTART,
    INSTALLER_PATH,
    WIN_INSTALLER_PATH,
    get_agent_logs,
    get_agent_version,
    get_latest_win_agent_version,
    get_win_agent_version,
    has_choco,
    is_agent_running_as_non_root,
    run_init_system_image,
    run_win_command,
    uninstall_win_agent,
    verify_override_files,
)

pytestmark = pytest.mark.installer

DEB_DISTROS = [
    ("debian-8-jessie", INIT_SYSTEMD),
    ("debian-9-stretch", INIT_SYSTEMD),
    ("ubuntu1404", INIT_UPSTART),
    ("ubuntu1604", INIT_SYSTEMD),
    ("ubuntu1804", INIT_SYSTEMD),
]

RPM_DISTROS = [
    ("amazonlinux1", INIT_UPSTART),
    ("amazonlinux2", INIT_SYSTEMD),
    ("centos7", INIT_SYSTEMD),
    ("centos8", INIT_SYSTEMD),
    ("opensuse15", INIT_SYSTEMD),
]

AGENT_VERSIONS = os.environ.get("AGENT_VERSIONS", "4.7.7,latest").split(",")
STAGE = os.environ.get("STAGE", "release")


@contextmanager
def _run_tests(base_image, init_system, installer_args, user=None, **extra_run_kwargs):
    if user:
        installer_args = f"--service-user {user} --service-group {user} {installer_args}"
    else:
        user = "signalfx-agent"
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
            verify_override_files(cont, init_system, user)
            assert is_agent_running_as_non_root(cont, user), f"Agent is not running as {user} user"
            yield backend, cont
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))


@pytest.mark.parametrize("base_image,init_system", DEB_DISTROS + RPM_DISTROS)
@pytest.mark.parametrize("agent_version", AGENT_VERSIONS)
def test_installer_on_all_distros(base_image, init_system, agent_version):
    if agent_version.endswith("-post") and (base_image, init_system) in RPM_DISTROS:
        agent_version = agent_version.replace("-post", "~post")
    elif agent_version.endswith("~post") and (base_image, init_system) in DEB_DISTROS:
        agent_version = agent_version.replace("~post", "-post")
    args = "MYTOKEN" if agent_version == "latest" else f"--package-version {agent_version}-1 MYTOKEN"
    args = args if STAGE == "release" else f"--{STAGE} {args}"
    if agent_version == "latest" or tuple(agent_version.split(".")) >= ("5", "1", "0"):
        user = "test-user"
    else:
        user = None
    with _run_tests(base_image, init_system, args, user=user) as [backend, cont]:
        if agent_version != "latest":
            installed_version = get_agent_version(cont)
            agent_version = agent_version.replace("~", "-")
            assert (
                installed_version == agent_version
            ), f"Installed agent version is {installed_version} but should be {agent_version}"
        try:
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"
        finally:
            print("\nDatapoints received:")
            for dp in backend.datapoints:
                print_dp_or_event(dp)
            print("\nEvents received:")
            for event in backend.events:
                print_dp_or_event(event)
            print(f"\nDimensions set: {backend.dims}")


def test_installer_different_realm():
    with _run_tests(
        "ubuntu1804",
        INIT_SYSTEMD,
        "MYTOKEN --realm us1",
        ingest_host="ingest.us1.signalfx.com",
        api_host="api.us1.signalfx.com",
    ) as [backend, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "host-metadata")), "Datapoints didn't come through"


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
            assert host is not None
            assert host in backend.dims["host"]
            dim = backend.dims["host"][host]
            assert dim["customProperties"] == {"cluster": "prod"}
            assert dim["tags"] in [None, []]

        wait_for_assertion(assert_cluster_property)


def run_win_installer(backend, args=""):
    installer_cmd = f'"{WIN_INSTALLER_PATH}" -ingest_url {backend.ingest_url} -api_url {backend.api_url} {args}'.strip()
    run_win_command(f"powershell {installer_cmd}", cwd="c:\\")
    return get_win_agent_version()


@pytest.mark.windows_only
@pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows")
@pytest.mark.parametrize("agent_version", AGENT_VERSIONS)
def test_win_zip_installer(agent_version):
    uninstall_win_agent()
    with ensure_fake_backend() as backend:
        try:
            args = f"-access_token MYTOKEN -stage {STAGE} -format zip"
            if agent_version == "latest":
                agent_version = get_latest_win_agent_version(stage=STAGE, agent_format="zip")
            else:
                args += f" -agent_version {agent_version}"
            installed_version = run_win_installer(backend, args)
            assert installed_version == agent_version, "installed agent version is '%s', expected '%s'" % (
                installed_version,
                agent_version,
            )
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"
        finally:
            print("\nDatapoints received:")
            for dp in backend.datapoints:
                print_dp_or_event(dp)
            print("\nEvents received:")
            for event in backend.events:
                print_dp_or_event(event)
            print(f"\nDimensions set: {backend.dims}")


@pytest.mark.windows_only
@pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows")
def test_win_local_msi_installer(request):
    # Get msi path from command line flag to pytest
    msi_path = request.config.getoption("--test-msi-path")
    if not msi_path:
        raise ValueError("You must specify the --test-msi-path flag to run msi tests")

    msi_path = os.path.abspath(msi_path)
    assert os.path.isfile(msi_path), f"{msi_path} not found!"

    uninstall_win_agent()

    with ensure_fake_backend() as backend:
        try:
            args = f"-access_token MYTOKEN -msi_path {msi_path}"
            run_win_installer(backend, args)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"
        finally:
            print("\nDatapoints received:")
            for dp in backend.datapoints:
                print_dp_or_event(dp)
            print("\nEvents received:")
            for event in backend.events:
                print_dp_or_event(event)
            print(f"\nDimensions set: {backend.dims}")
            uninstall_win_agent(msi_path=msi_path)


@pytest.mark.windows_only
@pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows")
def test_win_local_nupkg(request):
    assert has_choco(), "choco not installed"

    # Get nupkg path from command line flag to pytest
    nupkg_path = request.config.getoption("--test-nupkg-path")
    if not nupkg_path:
        raise ValueError("You must specify the --test-nupkg-path flag to run choco tests")

    nupkg_path = os.path.abspath(nupkg_path)
    assert os.path.isfile(nupkg_path), f"{nupkg_path} not found!"

    nupkg_dir = os.path.dirname(nupkg_path)

    uninstall_win_agent()

    with ensure_fake_backend() as backend:
        try:
            params = f"/access_token:MYTOKEN /ingest_url:{backend.ingest_url} /api_url:{backend.api_url}"
            run_win_command(f"choco install signalfx-agent -y -s {nupkg_dir} --params=\"'{params}'\"")
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"
        finally:
            print("\nDatapoints received:")
            for dp in backend.datapoints:
                print_dp_or_event(dp)
            print("\nEvents received:")
            for event in backend.events:
                print_dp_or_event(event)
            print(f"\nDimensions set: {backend.dims}")
            run_win_command("choco uninstall -y signalfx-agent")
            uninstall_win_agent()

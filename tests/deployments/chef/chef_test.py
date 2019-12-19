# Tests of the chef cookbook

import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from functools import partial as p
from pathlib import Path

import pytest

from tests.helpers.agent import ensure_fake_backend
from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.formatting import print_dp_or_event
from tests.helpers.util import print_lines, wait_for
from tests.packaging.common import (
    INIT_SYSTEMD,
    INIT_UPSTART,
    WIN_REPO_ROOT_DIR,
    get_agent_logs,
    get_agent_version,
    get_win_agent_version,
    has_choco,
    is_agent_running_as_non_root,
    run_init_system_image,
    run_win_command,
    running_in_azure_pipelines,
    uninstall_win_agent,
)
from tests.paths import REPO_ROOT_DIR

pytestmark = [pytest.mark.chef, pytest.mark.deployment]

ATTRIBUTES_JSON = """
{"signalfx_agent": {
  "package_stage": "final",
  "agent_version": null,
  "conf": {
    "signalFxAccessToken": "testing",
    "ingestUrl": "https://ingest.us0.signalfx.com",
    "apiUrl": "https://api.us0.signalfx.com",
    "observers": [
      {"type": "host"}
    ],
    "monitors": [
      {"type": "host-metadata"}
    ]
  }
}}
"""
CHEF_CMD = "chef-client -z -o 'recipe[signalfx_agent::default]' -j {0}"
SCRIPT_DIR = Path(__file__).parent.resolve()
DOCKERFILES_DIR = SCRIPT_DIR / "images"

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
    ("centos6", INIT_UPSTART),
    ("centos7", INIT_SYSTEMD),
    ("centos8", INIT_SYSTEMD),
    ("opensuse15", INIT_SYSTEMD),
]

# allow CHEF_VERSIONS env var with comma-separated chef versions for test parameterization
CHEF_VERSIONS = os.environ.get("CHEF_VERSIONS", "12.22.5,latest").split(",")

STAGE = os.environ.get("STAGE", "final")
INITIAL_VERSION = os.environ.get("INITIAL_VERSION", "4.7.7")
UPGRADE_VERSION = os.environ.get("UPGRADE_VERSION", "4.7.8")

WIN_CHEF_BIN_DIR = r"C:\opscode\chef\bin"
WIN_CHEF_COOKBOOKS_DIR = r"C:\chef\cookbooks"
WIN_AGENT_COOKBOOK_SRC_DIR = os.path.join(WIN_REPO_ROOT_DIR, "deployments", "chef")
WIN_AGENT_COOKBOOK_DEST_DIR = os.path.join(WIN_CHEF_COOKBOOKS_DIR, "signalfx_agent")
WINDOWS_COOKBOOK_DIR = os.path.join(WIN_CHEF_COOKBOOKS_DIR, "windows")
WINDOWS_COOKBOOK_URL = "https://supermarket.chef.io/cookbooks/windows/versions/6.0.0/download"


def run_chef_client(cont, chef_version, agent_version, stage):
    attributes = json.loads(ATTRIBUTES_JSON)
    attributes["signalfx_agent"]["agent_version"] = agent_version
    attributes["signalfx_agent"]["package_stage"] = stage
    print(attributes)
    with tempfile.NamedTemporaryFile(mode="w", dir="/tmp/scratch") as fd:
        fd.write(json.dumps(attributes))
        fd.flush()
        cmd = CHEF_CMD.format(fd.name)
        if chef_version == "latest" or int(chef_version.split(".")[0]) >= 15:
            cmd += " --chef-license accept-silent"
        print('running "%s" ...' % cmd)
        code, output = cont.exec_run(cmd)
        output = output.decode("utf-8").strip()
        assert code == 0, "failed to install agent:\n%s" % output
        print(output)
    installed_version = get_agent_version(cont)
    assert installed_version == agent_version, "installed agent version is '%s', expected '%s'" % (
        installed_version,
        agent_version,
    )
    assert is_agent_running_as_non_root(cont), "Agent is not running as non-root user"


@pytest.mark.parametrize(
    "base_image,init_system",
    [pytest.param(distro, init, marks=pytest.mark.deb) for distro, init in DEB_DISTROS]
    + [pytest.param(distro, init, marks=pytest.mark.rpm) for distro, init in RPM_DISTROS],
)
@pytest.mark.parametrize("chef_version", CHEF_VERSIONS)
def test_chef(base_image, init_system, chef_version):
    if base_image == "centos8" and chef_version != "latest" and int(chef_version.split(".")[0]) < 15:
        pytest.skip(f"chef {chef_version} not supported on centos 8")
    buildargs = {"CHEF_INSTALLER_ARGS": ""}
    if chef_version != "latest":
        buildargs["CHEF_INSTALLER_ARGS"] = f"-v {chef_version}"
    opts = {"path": REPO_ROOT_DIR, "dockerfile": DOCKERFILES_DIR / f"Dockerfile.{base_image}", "buildargs": buildargs}
    with run_init_system_image(base_image, **opts) as [cont, backend]:
        try:
            run_chef_client(cont, chef_version, INITIAL_VERSION, STAGE)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            # upgrade agent
            run_chef_client(cont, chef_version, UPGRADE_VERSION, STAGE)
            backend.reset_datapoints()
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            # downgrade agent for distros that support package downgrades
            if base_image not in ("debian-7-wheezy", "debian-8-jessie", "ubuntu1404"):
                run_chef_client(cont, chef_version, INITIAL_VERSION, STAGE)
                backend.reset_datapoints()
                assert wait_for(
                    p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
                ), "Datapoints didn't come through"
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))


def get_win_chef_version():
    output = run_win_command("chef-client --version").stdout.decode("utf-8")
    match = re.search(r"(\d+\.\d+\.\d+)", output)
    assert match and match.group(1).strip(), "failed to get version from output:\n%s" % output
    return match.group(1).strip()


def run_win_chef_client(backend, agent_version, stage):
    attributes = json.loads(ATTRIBUTES_JSON)
    attributes["signalfx_agent"]["agent_version"] = agent_version
    attributes["signalfx_agent"]["package_stage"] = stage
    attributes["signalfx_agent"]["conf"]["ingestUrl"] = backend.ingest_url
    attributes["signalfx_agent"]["conf"]["apiUrl"] = backend.api_url
    if running_in_azure_pipelines():
        attributes["signalfx_agent"]["user"] = os.environ.get("USERNAME")
        attributes["signalfx_agent"]["group"] = os.environ.get("USERNAME")
    print(attributes)
    attributes_path = r"C:\chef\cookbooks\attributes.json"
    with open(attributes_path, "w+") as fd:
        fd.write(json.dumps(attributes))
        fd.flush()
        if int(get_win_chef_version().split(".")[0]) >= 15:
            cmd = CHEF_CMD.format(attributes_path) + " --chef-license accept-silent"
        else:
            cmd = CHEF_CMD.format(attributes_path)
        print('running "%s" ...' % cmd)
        proc = subprocess.run(
            cmd, cwd=r"C:\chef\cookbooks", stdout=subprocess.PIPE, stderr=subprocess.STDOUT, shell=True, close_fds=False
        )
        output = proc.stdout.decode("utf-8")
        assert proc.returncode == 0, output
        print(output)
    installed_version = get_win_agent_version()
    assert installed_version == agent_version, "installed agent version is '%s', expected '%s'" % (
        installed_version,
        agent_version,
    )


def run_win_chef_setup(chef_version):
    assert has_choco(), "choco not installed!"
    uninstall_win_agent()
    if run_win_command("chef-client --version", []).returncode == 0:
        run_win_command("choco uninstall -y -f chef-client")
    if chef_version == "latest":
        run_win_command(f"choco upgrade -y -f chef-client")
    else:
        run_win_command(f"choco upgrade -y -f chef-client --version {chef_version}")
    if WIN_CHEF_BIN_DIR not in os.environ.get("PATH"):
        os.environ["PATH"] = WIN_CHEF_BIN_DIR + ";" + os.environ.get("PATH")
    os.makedirs(WIN_CHEF_COOKBOOKS_DIR, exist_ok=True)
    if os.path.isdir(WIN_AGENT_COOKBOOK_DEST_DIR):
        shutil.rmtree(WIN_AGENT_COOKBOOK_DEST_DIR)
    shutil.copytree(WIN_AGENT_COOKBOOK_SRC_DIR, WIN_AGENT_COOKBOOK_DEST_DIR)
    if not os.path.isdir(WINDOWS_COOKBOOK_DIR):
        run_win_command(
            f'powershell -command "curl -outfile windows.tar.gz {WINDOWS_COOKBOOK_URL}"', cwd=WIN_CHEF_COOKBOOKS_DIR
        )
        run_win_command('powershell -command "tar -zxvf windows.tar.gz"', cwd=WIN_CHEF_COOKBOOKS_DIR)


@pytest.mark.windows_only
@pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows")
@pytest.mark.parametrize("chef_version", CHEF_VERSIONS)
def test_chef_on_windows(chef_version):
    run_win_chef_setup(chef_version)
    with ensure_fake_backend() as backend:
        try:
            run_win_chef_client(backend, INITIAL_VERSION, STAGE)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            # upgrade agent
            run_win_chef_client(backend, UPGRADE_VERSION, STAGE)
            backend.reset_datapoints()
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            # downgrade agent
            run_win_chef_client(backend, INITIAL_VERSION, STAGE)
            backend.reset_datapoints()
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

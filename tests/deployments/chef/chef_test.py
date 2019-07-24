# Tests of the chef cookbook

import json
import os
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
    get_agent_logs,
    get_agent_version,
    get_win_agent_version,
    is_agent_running_as_non_root,
    run_init_system_image,
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
    ("opensuse15", INIT_SYSTEMD),
]

# allow CHEF_VERSIONS env var with comma-separated chef versions for test parameterization
CHEF_VERSIONS = os.environ.get("CHEF_VERSIONS", "latest").split(",")

STAGE = os.environ.get("STAGE", "final")
INITIAL_VERSION = os.environ.get("INITIAL_VERSION", "4.7.7")
UPGRADE_VERSION = os.environ.get("UPGRADE_VERSION", "4.7.8")


def run_chef_client(cont, chef_version, agent_version=None, stage=STAGE):
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
    assert is_agent_running_as_non_root(cont), "Agent is not running as non-root user"
    return get_agent_version(cont)


@pytest.mark.parametrize(
    "base_image,init_system",
    [pytest.param(distro, init, marks=pytest.mark.deb) for distro, init in DEB_DISTROS]
    + [pytest.param(distro, init, marks=pytest.mark.rpm) for distro, init in RPM_DISTROS],
)
@pytest.mark.parametrize("chef_version", CHEF_VERSIONS)
def test_chef(base_image, init_system, chef_version):
    dockerfile = DOCKERFILES_DIR / f"Dockerfile.{base_image}"
    buildargs = {"CHEF_INSTALLER_ARGS": ""}
    if chef_version != "latest":
        buildargs["CHEF_INSTALLER_ARGS"] = f"-v {chef_version}"
    with run_init_system_image(base_image, path=REPO_ROOT_DIR, dockerfile=dockerfile, buildargs=buildargs) as [
        cont,
        backend,
    ]:
        try:
            installed_version = run_chef_client(cont, chef_version, INITIAL_VERSION, STAGE).replace("-", "~")
            assert installed_version == INITIAL_VERSION, "installed agent version is '%s', expected '%s'" % (
                installed_version,
                INITIAL_VERSION,
            )
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            # upgrade agent
            installed_version = run_chef_client(cont, chef_version, UPGRADE_VERSION, STAGE).replace("-", "~")
            assert installed_version == UPGRADE_VERSION, "installed agent version is '%s', expected '%s'" % (
                installed_version,
                UPGRADE_VERSION,
            )
            backend.reset_datapoints()
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"

            # downgrade agent for distros that support package downgrades
            if base_image not in ("debian-7-wheezy", "debian-8-jessie", "ubuntu1404"):
                installed_version = run_chef_client(cont, chef_version, INITIAL_VERSION, STAGE).replace("-", "~")
                assert installed_version == INITIAL_VERSION, "installed agent version is '%s', expected '%s'" % (
                    installed_version,
                    INITIAL_VERSION,
                )
                backend.reset_datapoints()
                assert wait_for(
                    p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
                ), "Datapoints didn't come through"
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))


def run_win_chef_client(backend, agent_version=None, stage=STAGE):
    attributes = json.loads(ATTRIBUTES_JSON)
    attributes["signalfx_agent"]["agent_version"] = agent_version
    attributes["signalfx_agent"]["package_stage"] = stage
    attributes["signalfx_agent"]["conf"]["ingestUrl"] = backend.ingest_url
    attributes["signalfx_agent"]["conf"]["apiUrl"] = backend.api_url
    if os.environ.get("AZURE_HTTP_USER_AGENT"):
        # running in Azure Pipelines; need to override Administrator user/group
        attributes["signalfx_agent"]["user"] = os.environ.get("USERNAME")
        attributes["signalfx_agent"]["group"] = os.environ.get("USERNAME")
    print(attributes)
    attributes_path = r"C:\chef\cookbooks\attributes.json"
    with open(attributes_path, "w+") as fd:
        fd.write(json.dumps(attributes))
        fd.flush()
        cmd = CHEF_CMD.format(attributes_path) + " --chef-license accept-silent"
        print('running "%s" ...' % cmd)
        proc = subprocess.run(
            cmd, cwd=r"C:\chef\cookbooks", stdout=subprocess.PIPE, stderr=subprocess.STDOUT, shell=True
        )
        output = proc.stdout.decode("utf-8")
        assert proc.returncode == 0, output
        print(output)
    installed_version = get_win_agent_version()
    if agent_version:
        assert installed_version == agent_version, "installed agent version is '%s', expected '%s'" % (
            installed_version,
            agent_version,
        )
    return installed_version


@pytest.mark.windows_only
@pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows")
def test_chef_on_windows():
    with ensure_fake_backend() as backend:
        try:
            run_win_chef_client(backend, INITIAL_VERSION)
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"
            run_win_chef_client(backend, UPGRADE_VERSION)
            backend.reset_datapoints()
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "host-metadata")
            ), "Datapoints didn't come through"
            run_win_chef_client(backend, INITIAL_VERSION)
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

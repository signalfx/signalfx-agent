import json
import os
import pytest
import semver
import subprocess
import urllib.request

SCRIPTS_DIR = os.path.join(os.path.dirname(os.path.realpath(__file__)), "..", "scripts")
MIN_K8S_VERSION = 'v1.7.0'
K8S_DEFAULT_TIMEOUT = 300
DEFAULT_AGENT_IMAGE_NAME = "quay.io/signalfx/signalfx-agent-dev"
try:
    proc = subprocess.run(os.path.join(SCRIPTS_DIR, "current-version"), shell=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    DEFAULT_AGENT_IMAGE_TAG = proc.stdout.decode('utf-8').strip()
except:
    DEFAULT_AGENT_IMAGE_TAG = "latest"

def get_k8s_supported_versions():
    with urllib.request.urlopen('https://storage.googleapis.com/minikube/k8s_releases.json') as f:
        k8s_releases_json = json.loads(f.read().decode('utf-8'))
    versions = []
    for r in k8s_releases_json:
        if semver.match(r['version'].strip('v'), '>=' + MIN_K8S_VERSION.strip('v')):
            versions.append(r['version'])
    return versions

K8S_SUPPORTED_VERSIONS = get_k8s_supported_versions()

def pytest_addoption(parser):
    parser.addoption(
        "--k8s_version",
        action="store",
        default=K8S_SUPPORTED_VERSIONS[0],
        help="Comma-separated kubernetes versions for minikube to deploy (default=%s). Use '--k8s_versions=all' to test all supported versions" % K8S_SUPPORTED_VERSIONS[0]
    )
    parser.addoption(
        "--k8s_timeout",
        action="store",
        default=K8S_DEFAULT_TIMEOUT,
        help="Timeout (in seconds) to wait for the minikube cluster to be ready (default=%d)" % K8S_DEFAULT_TIMEOUT
    )
    parser.addoption(
        "--agent_name",
        action="store",
        default=DEFAULT_AGENT_IMAGE_NAME,
        help="SignalFx agent image name (default=%s)" % DEFAULT_AGENT_IMAGE_NAME
    )
    parser.addoption(
        "--agent_tag",
        action="store",
        default=DEFAULT_AGENT_IMAGE_TAG,
        help="SignalFx agent image tag (default=%s)" % DEFAULT_AGENT_IMAGE_TAG
    )

def pytest_generate_tests(metafunc):
    if 'k8s_version' in metafunc.fixturenames:
        k8s_version = metafunc.config.getoption("k8s_version")
        if k8s_version:
            if k8s_version.lower() == "all":
                metafunc.parametrize("k8s_version", K8S_SUPPORTED_VERSIONS)
            else:
                for k in k8s_version.split(','):
                    assert k in K8S_SUPPORTED_VERSIONS, "k8s_version value \"%s\" not supported!" % k8s_version
                metafunc.parametrize("k8s_version", k8s_version.split(','))

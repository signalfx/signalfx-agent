import json
import os
import pytest
import semver
import subprocess
import sys
import time
import urllib.request

SCRIPTS_DIR = os.path.join(os.path.dirname(os.path.realpath(__file__)), "..", "scripts")
K8S_MIN_VERSION = 'v1.7.0'
K8S_DEFAULT_TIMEOUT = 300
K8S_DEFAULT_METRICS_TIMEOUT = 300
K8S_DEFAULT_AGENT_IMAGE_NAME = "quay.io/signalfx/signalfx-agent-dev"
try:
    proc = subprocess.run(os.path.join(SCRIPTS_DIR, "current-version"), shell=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    K8S_DEFAULT_AGENT_IMAGE_TAG = proc.stdout.decode('utf-8').strip()
except:
    K8S_DEFAULT_AGENT_IMAGE_TAG = "latest"

def get_k8s_supported_versions():
    k8s_releases_json = None
    attempt = 0
    while attempt < 3:
        try:
            with urllib.request.urlopen('https://storage.googleapis.com/minikube/k8s_releases.json') as f:
                k8s_releases_json = json.loads(f.read().decode('utf-8'))
            break
        except:
            time.sleep(5)
            k8s_releases_json = None
            attempt += 1
    if not k8s_releases_json:
        print("Failed to get K8S releases from https://storage.googleapis.com/minikube/k8s_releases.json !")
        sys.exit(1)
    versions = []
    for r in k8s_releases_json:
        if semver.match(r['version'].strip('v'), '>=' + K8S_MIN_VERSION.strip('v')):
            versions.append(r['version'])
    return versions

K8S_SUPPORTED_VERSIONS = get_k8s_supported_versions()

def pytest_addoption(parser):
    parser.addoption(
        "--k8s_versions",
        action="store",
        default=K8S_SUPPORTED_VERSIONS[0],
        help="Comma-separated K8S versions for minikube to deploy (default=%s). Use '--k8s_versions=all' to test all supported versions." % K8S_SUPPORTED_VERSIONS[0]
    )
    parser.addoption(
        "--k8s_timeout",
        action="store",
        default=K8S_DEFAULT_TIMEOUT,
        help="Timeout (in seconds) to wait for the minikube cluster to be ready (default=%d)." % K8S_DEFAULT_TIMEOUT
    )
    parser.addoption(
        "--k8s_agent_name",
        action="store",
        default=K8S_DEFAULT_AGENT_IMAGE_NAME,
        help="SignalFx agent image name to use for K8S tests (default=%s). The image must exist either locally or remotely." % K8S_DEFAULT_AGENT_IMAGE_NAME
    )
    parser.addoption(
        "--k8s_agent_tag",
        action="store",
        default=K8S_DEFAULT_AGENT_IMAGE_TAG,
        help="SignalFx agent image tag to use for K8S tests (default=%s). The image must exist either locally or remotely." % K8S_DEFAULT_AGENT_IMAGE_TAG
    )
    parser.addoption(
        "--k8s_metrics_timeout",
        action="store",
        default=K8S_DEFAULT_METRICS_TIMEOUT,
        help="Timeout (in seconds) for K8S metrics tests (default=%d)." % K8S_DEFAULT_METRICS_TIMEOUT
    )

def pytest_generate_tests(metafunc):
    if 'k8s_version' in metafunc.fixturenames:
        k8s_versions = metafunc.config.getoption("k8s_versions")
        if k8s_versions:
            if k8s_versions.lower() == "all":
                metafunc.parametrize("k8s_version", K8S_SUPPORTED_VERSIONS)
            else:
                for v in k8s_versions.split(','):
                    assert v in K8S_SUPPORTED_VERSIONS, "K8S version \"%s\" not supported!" % v
                metafunc.parametrize("k8s_version", k8s_versions.split(','))

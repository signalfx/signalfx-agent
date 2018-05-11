import itertools
import json
import os
import pytest
import semver
import subprocess
import sys
import time
import urllib.request

SCRIPTS_DIR = os.path.join(os.path.dirname(os.path.realpath(__file__)), "..", "scripts")
K8S_MIN_VERSION = '1.7.0'
K8S_MAX_VERSION = '1.9.4'
K8S_DEFAULT_TIMEOUT = 300
K8S_DEFAULT_TEST_TIMEOUT = 120
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
        version = r['version'].strip().strip('v')
        if semver.match(version, '>=' + K8S_MIN_VERSION) and semver.match(version, '<=' + K8S_MAX_VERSION):
            versions.append(version)
    return sorted(versions, key=lambda v: list(map(int, v.split('.'))), reverse=True)

K8S_SUPPORTED_VERSIONS = get_k8s_supported_versions()
K8S_MAJOR_MINOR_VERSIONS = [v for v in K8S_SUPPORTED_VERSIONS if semver.parse_version_info(v).patch == 0]

K8S_SUPPORTED_OBSERVERS = ["k8s-api", "k8s-kubelet"]
K8S_DEFAULT_OBSERVER = "k8s-api"


def pytest_addoption(parser):
    parser.addoption(
        "--k8s-versions",
        action="store",
        default=K8S_MAJOR_MINOR_VERSIONS[0],
        help="Comma-separated string of K8S cluster versions for minikube to deploy (default=%s). Use '--k8s-versions=all' to test all supported versions. Use '--k8s-versions=minor' to test all supported non-patch versions (e.g. 1.7.0, 1.8.0, 1.9.0, etc.). This option is ignored if the --k8s-container option also is specified." % K8S_MAJOR_MINOR_VERSIONS[0]
    )
    parser.addoption(
        "--k8s-observers",
        action="store",
        default=K8S_DEFAULT_OBSERVER,
        help="Comma-separated string of observers for the SignalFx agent (default=%s). Use '--k8s-observers=all' to test all supported observers." % K8S_DEFAULT_OBSERVER
    )
    parser.addoption(
        "--k8s-timeout",
        action="store",
        default=K8S_DEFAULT_TIMEOUT,
        help="Timeout (in seconds) to wait for the minikube cluster to be ready (default=%d)." % K8S_DEFAULT_TIMEOUT
    )
    parser.addoption(
        "--k8s-agent-name",
        action="store",
        default=K8S_DEFAULT_AGENT_IMAGE_NAME,
        help="SignalFx agent image name to use for K8S tests (default=%s). The image must exist either locally or remotely." % K8S_DEFAULT_AGENT_IMAGE_NAME
    )
    parser.addoption(
        "--k8s-agent-tag",
        action="store",
        default=K8S_DEFAULT_AGENT_IMAGE_TAG,
        help="SignalFx agent image tag to use for K8S tests (default=%s). The image must exist either locally or remotely." % K8S_DEFAULT_AGENT_IMAGE_TAG
    )
    parser.addoption(
        "--k8s-test-timeout",
        action="store",
        default=K8S_DEFAULT_TEST_TIMEOUT,
        help="Timeout (in seconds) for each K8S test (default=%d)." % K8S_DEFAULT_TEST_TIMEOUT
    )
    parser.addoption(
        "--k8s-container",
        action="store",
        default=None,
        help="Name of a running minikube container to use for the tests (the container should not have the agent or any services already running). If not specified, a new minikube container will automatically be deployed."
    )
    parser.addoption(
        "--k8s-skip-teardown",
        action="store_true",
        help="If specified, the minikube container will not be stopped/removed when the tests complete."
    )


def pytest_generate_tests(metafunc):
    if 'minikube' in metafunc.fixturenames:
        k8s_versions = metafunc.config.getoption("--k8s-versions")
        versions_to_test = []
        if not k8s_versions:
            versions_to_test = [K8S_MAJOR_MINOR_VERSIONS[0]]
        elif k8s_versions.lower() == "latest":
            versions_to_test = [K8S_SUPPORTED_VERSIONS[0]]
        elif k8s_versions.lower() == "all":
            versions_to_test = K8S_SUPPORTED_VERSIONS
        elif k8s_versions.lower() == "minor":
            versions_to_test = K8S_MAJOR_MINOR_VERSIONS
        else:
            for v in k8s_versions.split(','):
                assert v.strip('v') in K8S_SUPPORTED_VERSIONS, "K8S version \"%s\" not supported!" % v
            versions_to_test = k8s_versions.split(',')
        metafunc.parametrize("minikube", versions_to_test, ids=["v%s" % v.strip('v') for v in versions_to_test], scope="module", indirect=True)
    if 'k8s_observer' in metafunc.fixturenames:
        k8s_observers = metafunc.config.getoption("--k8s-observers")
        if not k8s_observers:
            observers_to_test = [K8S_DEFAULT_OBSERVER]
        elif k8s_observers.lower() == 'all':
            observers_to_test = K8S_SUPPORTED_OBSERVERS
        else:
            for o in k8s_observers.split(','):
                assert o in K8S_SUPPORTED_OBSERVERS, "observer \"%s\" not supported!" % o
            observers_to_test = k8s_observers.split(',')
        metafunc.parametrize("k8s_observer", observers_to_test, ids=[o for o in observers_to_test], indirect=True)


import json
import pytest
import semver
import urllib.request

MIN_K8S_VERSION = 'v1.7.0'

def get_k8s_supported_versions():
    with urllib.request.urlopen('https://storage.googleapis.com/minikube/k8s_releases.json') as f:
        k8s_releases_json = json.loads(f.read().decode('utf-8'))
    versions = []
    for r in k8s_releases_json:
        if semver.match(r['version'].strip('v'), '>=' + MIN_K8S_VERSION.strip('v')):
            versions.append(r['version'])
    return versions

k8s_supported_versions = get_k8s_supported_versions()
k8s_default_timeout = 300

def pytest_addoption(parser):
    parser.addoption(
        "--k8s_version",
        action="store",
        default=k8s_supported_versions[0],
        help="Comma-separated kubernetes versions for minikube to deploy (default=%s). Use '--k8s_versions=all' to test all supported versions" % k8s_supported_versions[0]
    )
    parser.addoption(
        "--k8s_timeout",
        action="store",
        default=k8s_default_timeout,
        help="Timeout (in seconds) to wait for the minikube cluster to be ready (default=%d)" % k8s_default_timeout
    )

def pytest_generate_tests(metafunc):
    if 'k8s_version' in metafunc.fixturenames:
        k8s_version = metafunc.config.getoption("k8s_version")
        if k8s_version:
            if k8s_version.lower() == "all":
                metafunc.parametrize("k8s_version", k8s_supported_versions)
            else:
                for k in k8s_version.split(','):
                    assert k in k8s_supported_versions, "k8s_version value \"%s\" not supported!" % k8s_version
                metafunc.parametrize("k8s_version", k8s_version.split(','))

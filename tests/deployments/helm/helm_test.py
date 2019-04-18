# Tests of the helm chart

import os
import tempfile
from functools import partial as p

import pytest
import yaml

from tests.helpers import fake_backend
from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.kubernetes.utils import create_clusterrolebinding, daemonset_is_ready, deployment_is_ready
from tests.helpers.util import get_host_ip, wait_for
from tests.packaging.common import PROJECT_DIR, copy_file_into_container

LOCAL_CHART_DIR = os.path.join(PROJECT_DIR, "deployments/k8s/helm/signalfx-agent")
CONTAINER_CHART_DIR = "/opt/deployments/k8s/helm/signalfx-agent"
CUR_DIR = os.path.abspath(os.path.dirname(__file__))
NGINX_YAML_PATH = os.path.join(PROJECT_DIR, "tests/monitors/nginx/nginx-k8s.yaml")
CLUSTERROLEBINDING_YAML_PATH = os.path.join(CUR_DIR, "clusterrolebinding.yaml")
MONITORS_CONFIG = """
    - type: host-metadata
    - type: collectd/nginx
      discoveryRule: container_image =~ "nginx" && private_port == 80
      url: "http://{{.Host}}:{{.Port}}/nginx_status"
"""

pytestmark = [pytest.mark.helm, pytest.mark.deployment]


def create_cluster_admin_rolebinding(minikube):
    clusterrolebinding_yaml = yaml.load(open(CLUSTERROLEBINDING_YAML_PATH).read())
    name = clusterrolebinding_yaml.get("metadata", {}).get("name")
    assert name, "name not found in %s" % CLUSTERROLEBINDING_YAML_PATH
    print("Creating %s cluster role binding ..." % name)
    create_clusterrolebinding(body=clusterrolebinding_yaml)
    minikube.exec_kubectl("describe clusterrolebinding %s" % name)


def update_values_yaml(minikube, backend, namespace):
    values_yaml = None
    with open(os.path.join(LOCAL_CHART_DIR, "values.yaml")) as fd:
        values_yaml = yaml.load(fd.read())
        values_yaml["signalFxAccessToken"] = "testing123"
        values_yaml["clusterName"] = minikube.cluster_name
        values_yaml["namespace"] = namespace
        values_yaml["ingestUrl"] = "http://%s:%d" % (backend.ingest_host, backend.ingest_port)
        values_yaml["apiUrl"] = "http://%s:%d" % (backend.api_host, backend.api_port)
        values_yaml["monitors"] = yaml.load(MONITORS_CONFIG)
    with tempfile.NamedTemporaryFile(mode="w") as fd:
        fd.write(yaml.dump(values_yaml))
        fd.flush()
        copy_file_into_container(fd.name, minikube.container, os.path.join(CONTAINER_CHART_DIR, "values.yaml"))


def init_helm(minikube):
    minikube.exec_cmd("helm init")
    print("Waiting for tiller-deployment to be ready ...")
    assert wait_for(
        p(deployment_is_ready, "tiller-deploy", "kube-system"), timeout_seconds=30, interval_seconds=2
    ), "timed out waiting for tiller-deployment to be ready!"


def get_chart_name_version():
    chart_path = os.path.join(LOCAL_CHART_DIR, "Chart.yaml")
    chart_name = None
    chart_version = None
    with open(chart_path) as fd:
        chart_yaml = yaml.load(fd.read())
        chart_name = chart_yaml.get("name")
        chart_version = chart_yaml.get("version")
    assert chart_name, "failed to get chart name from %s" % chart_path
    assert chart_version, "failed to get chart version from %s" % chart_path
    return chart_name, chart_version


def get_daemonset_name(minikube, namespace):
    chart_name, chart_version = get_chart_name_version()
    chart_release_name = chart_name + "-" + chart_version
    output = minikube.exec_cmd("helm list --namespace=%s --output=yaml" % namespace)
    release = None
    for rel in yaml.load(output).get("Releases", []):
        if rel.get("Chart") == chart_release_name:
            release = rel
            break
    assert release, "chart '%s' not found in helm list output:\n%s" % (chart_release_name, output)
    release_name = release.get("Name")
    assert release_name, "failed to get name for release:\n%s" % yaml.dump(release)
    return release_name + "-" + chart_name


def install_helm_chart(minikube, namespace):
    minikube.exec_cmd("helm install --namespace=%s --debug %s" % (namespace, CONTAINER_CHART_DIR))
    try:
        daemonset_name = get_daemonset_name(minikube, namespace)
        print("Waiting for daemonset %s to be ready ..." % daemonset_name)
        assert wait_for(p(daemonset_is_ready, daemonset_name, namespace), timeout_seconds=120, interval_seconds=2), (
            "timed out waiting for %s daemonset to be ready!" % daemonset_name
        )
    finally:
        minikube.exec_kubectl("get all --all-namespaces")


def test_helm(minikube, k8s_namespace):
    with minikube.create_resources([NGINX_YAML_PATH], namespace=k8s_namespace):
        with fake_backend.start(ip_addr=get_host_ip()) as backend:
            create_cluster_admin_rolebinding(minikube)
            init_helm(minikube)
            update_values_yaml(minikube, backend, k8s_namespace)
            install_helm_chart(minikube, k8s_namespace)
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "nginx"))
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata"))

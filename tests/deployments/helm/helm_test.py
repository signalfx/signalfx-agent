# Tests of the helm chart

import os
import subprocess
import tempfile
from contextlib import contextmanager
from functools import partial as p
from pathlib import Path

import pytest
import yaml
from kubernetes import client as kube_client
from tests.helpers import fake_backend
from tests.helpers.assertions import has_datapoint
from tests.helpers.kubernetes.utils import daemonset_is_ready, deployment_is_ready
from tests.helpers.util import wait_for
from tests.packaging.common import DEPLOYMENTS_DIR
from tests.paths import TEST_SERVICES_DIR

LOCAL_CHART_DIR = DEPLOYMENTS_DIR / "k8s/helm/signalfx-agent"
SCRIPT_DIR = Path(__file__).parent.resolve()
NGINX_YAML_PATH = TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"
MONITORS_CONFIG = """
    - type: host-metadata
    - type: collectd/nginx
      discoveryRule: container_image =~ "nginx" && private_port == 80
      url: "http://{{.Host}}:{{.Port}}/nginx_status"
"""

CLUSTER_ROLEBINDING_YAML = """
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: tiller-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: tiller
  namespace: NAMESPACE
"""

SERVICE_ACCOUNT_YAML = """
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
"""

pytestmark = [pytest.mark.helm, pytest.mark.deployment]


def helm_command_prefix(k8s_cluster):
    context_flag = ""
    if k8s_cluster.kube_context:
        context_flag = f"--kube-context {k8s_cluster.kube_context}"
    return (
        f"helm --kubeconfig {k8s_cluster.kube_config_path} {context_flag} "
        f"--tiller-namespace {k8s_cluster.test_namespace}"
    )


@contextmanager
def tiller_rbac_resources(k8s_cluster):
    print("Creating tiller RBAC resources ...")

    corev1 = kube_client.CoreV1Api()
    serviceaccount = corev1.create_namespaced_service_account(
        body=yaml.safe_load(SERVICE_ACCOUNT_YAML), namespace=k8s_cluster.test_namespace
    )

    rbacv1beta1 = kube_client.RbacAuthorizationV1beta1Api()

    clusterrolebinding_yaml = yaml.safe_load(CLUSTER_ROLEBINDING_YAML)
    clusterrolebinding_yaml["subjects"][0]["namespace"] = k8s_cluster.test_namespace
    clusterrolebinding_yaml["metadata"]["name"] += f"-{k8s_cluster.test_namespace}"
    clusterrolebinding = rbacv1beta1.create_cluster_role_binding(body=clusterrolebinding_yaml)
    try:
        yield
    finally:
        rbacv1beta1.delete_cluster_role_binding(
            name=clusterrolebinding.metadata.name, body=kube_client.V1DeleteOptions()
        )
        corev1.delete_namespaced_service_account(
            name=serviceaccount.metadata.name,
            namespace=serviceaccount.metadata.namespace,
            body=kube_client.V1DeleteOptions(),
        )


@contextmanager
def release_values_yaml(k8s_cluster, proxy_pod_ip, fake_services):
    image, tag = k8s_cluster.agent_image_name.rsplit(":", 1)
    values_yaml = {
        "signalFxAccessToken": "testing123",
        "image": {"repository": image, "tag": tag},
        "clusterName": k8s_cluster.test_namespace,
        "namespace": k8s_cluster.test_namespace,
        "ingestUrl": f"http://{proxy_pod_ip}:{fake_services.ingest_port}",
        "apiUrl": f"http://{proxy_pod_ip}:{fake_services.api_port}",
        "monitors": yaml.safe_load(MONITORS_CONFIG),
    }

    values_path = None
    with tempfile.NamedTemporaryFile(mode="w", delete=False) as fd:
        fd.write(yaml.dump(values_yaml))
        values_path = fd.name

    try:
        yield values_path
    finally:
        os.remove(values_path)


def init_helm(k8s_cluster):
    init_command = helm_command_prefix(k8s_cluster) + " init --service-account tiller"
    print(f"Executing helm init: {init_command}")
    output = subprocess.check_output(init_command, shell=True)
    print(f"Helm init output: {output}")

    print("Waiting for tiller-deployment to be ready ...")
    assert wait_for(
        p(deployment_is_ready, "tiller-deploy", k8s_cluster.test_namespace), timeout_seconds=45, interval_seconds=2
    ), "timed out waiting for tiller-deployment to be ready!"


def get_chart_name_version():
    chart_path = os.path.join(LOCAL_CHART_DIR, "Chart.yaml")
    chart_name = None
    chart_version = None
    with open(chart_path) as fd:
        chart_yaml = yaml.safe_load(fd.read())
        chart_name = chart_yaml.get("name")
        chart_version = chart_yaml.get("version")
    assert chart_name, "failed to get chart name from %s" % chart_path
    assert chart_version, "failed to get chart version from %s" % chart_path
    return chart_name, chart_version


def get_daemonset_name(k8s_cluster):
    chart_name, chart_version = get_chart_name_version()
    chart_release_name = chart_name + "-" + chart_version
    output = subprocess.check_output(
        helm_command_prefix(k8s_cluster) + f" list --namespace={k8s_cluster.test_namespace} --output=yaml", shell=True
    )
    release = None
    for rel in yaml.safe_load(output).get("Releases", []):
        if rel.get("Chart") == chart_release_name:
            release = rel
            break
    assert release, "chart '%s' not found in helm list output:\n%s" % (chart_release_name, output)
    release_name = release.get("Name")
    assert release_name, "failed to get name for release:\n%s" % yaml.dump(release)
    return release_name + "-" + chart_name


def install_helm_chart(k8s_cluster, values_path):
    install_cmd = (
        helm_command_prefix(k8s_cluster)
        + f" install --values {values_path} --namespace={k8s_cluster.test_namespace} --debug {LOCAL_CHART_DIR}",
    )
    print(f"Running Helm install: {install_cmd}")
    output = subprocess.check_output(install_cmd, shell=True)
    print(f"Helm chart install output: {output}")

    try:
        daemonset_name = get_daemonset_name(k8s_cluster)
        print("Waiting for daemonset %s to be ready ..." % daemonset_name)
        assert wait_for(
            p(daemonset_is_ready, daemonset_name, k8s_cluster.test_namespace), timeout_seconds=120, interval_seconds=2
        ), ("timed out waiting for %s daemonset to be ready!" % daemonset_name)
    finally:
        k8s_cluster.exec_kubectl("get all --all-namespaces")


def test_helm(k8s_cluster):
    with k8s_cluster.create_resources([NGINX_YAML_PATH]), tiller_rbac_resources(
        k8s_cluster
    ), fake_backend.start() as backend:
        init_helm(k8s_cluster)

        with k8s_cluster.run_tunnels(backend) as proxy_pod_ip:
            with release_values_yaml(k8s_cluster, proxy_pod_ip, backend) as values_path:
                install_helm_chart(k8s_cluster, values_path)
                assert wait_for(p(has_datapoint, backend, dimensions={"plugin": "nginx"}))
                assert wait_for(p(has_datapoint, backend, dimensions={"plugin": "signalfx-metadata"}))

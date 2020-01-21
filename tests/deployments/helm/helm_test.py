# Tests of the helm chart

import os
import string
import tempfile
from contextlib import contextmanager
from functools import partial as p
from pathlib import Path

import pytest
import yaml
from kubernetes import client as kube_client
from tests.helpers import fake_backend
from tests.helpers.assertions import has_datapoint
from tests.helpers.formatting import print_dp_or_event
from tests.helpers.kubernetes.agent import AGENT_STATUS_COMMAND
from tests.helpers.kubernetes.utils import (
    daemonset_is_ready,
    deployment_is_ready,
    exec_pod_command,
    get_pod_logs,
    get_pods_by_labels,
)
from tests.helpers.util import copy_file_content_into_container, copy_file_into_container, run_service, wait_for
from tests.packaging.common import DEPLOYMENTS_DIR
from tests.paths import REPO_ROOT_DIR, TEST_SERVICES_DIR

LOCAL_CHART_DIR = DEPLOYMENTS_DIR / "k8s/helm/signalfx-agent"
CONTAINER_CHART_DIR = "/opt/signalfx-agent"
SCRIPT_DIR = Path(__file__).parent.resolve()
APP_YAML_PATH = TEST_SERVICES_DIR / "prometheus/prometheus-k8s.yaml"
MONITORS_CONFIG = string.Template(
    """
    - type: prometheus/prometheus
      discoveryRule: kubernetes_pod_name =~ "prometheus-deployment" && kubernetes_namespace == "$namespace"
      sendAllMetrics: true
"""
)

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


def helm_command_prefix(k8s_cluster, helm_major_version):
    cmd = f"helm --kubeconfig {k8s_cluster.kube_config_path}"
    if k8s_cluster.kube_context:
        cmd = f"{cmd} --kube-context {k8s_cluster.kube_context}"
    if helm_major_version < 3:
        cmd = f"{cmd} --tiller-namespace {k8s_cluster.test_namespace}"
    return cmd


def exec_helm(cont, args):
    code, output = cont.exec_run(args)
    output = output.decode("utf-8")
    assert code == 0, f"{args}:\n{output}"
    return output


@contextmanager
def tiller_rbac_resources(k8s_cluster, helm_major_version):
    if helm_major_version < 3:
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
    else:
        yield


@contextmanager
def release_values_yaml(k8s_cluster, proxy_pod_ip, fake_services):
    image, tag = k8s_cluster.agent_image_name.rsplit(":", 1)
    values_yaml = {
        "signalFxAccessToken": "testing123",
        "image": {"repository": image, "tag": tag},
        "clusterName": k8s_cluster.test_namespace,
        "namespace": k8s_cluster.test_namespace,
        "ingestUrl": f"http://{proxy_pod_ip}:{fake_services.ingest_port}",
        "globalDimensions": {"application": "helm-test"},
        "apiUrl": f"http://{proxy_pod_ip}:{fake_services.api_port}",
        "monitors": yaml.safe_load(MONITORS_CONFIG.substitute(namespace=k8s_cluster.test_namespace)),
    }

    values_path = None
    with tempfile.NamedTemporaryFile(mode="w", delete=False) as fd:
        fd.write(yaml.dump(values_yaml))
        values_path = fd.name

    try:
        yield values_path
    finally:
        os.remove(values_path)


def init_helm(k8s_cluster, cont, helm_major_version):
    init_command = helm_command_prefix(k8s_cluster, helm_major_version) + " init --service-account tiller"
    print(f"Executing helm init: {init_command}")
    output = exec_helm(cont, init_command)
    print(f"Helm init output:\n{output}")

    print("Waiting for tiller-deployment to be ready ...")
    assert wait_for(
        p(deployment_is_ready, "tiller-deploy", k8s_cluster.test_namespace), timeout_seconds=90, interval_seconds=2
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


def get_daemonset_name(k8s_cluster, cont, helm_major_version):
    chart_name, chart_version = get_chart_name_version()
    chart_release_name = chart_name + "-" + chart_version
    list_cmd = (
        helm_command_prefix(k8s_cluster, helm_major_version)
        + f" list --namespace={k8s_cluster.test_namespace} --output=yaml"
    )
    print(f"Executing helm list: {list_cmd}")
    output = exec_helm(cont, list_cmd)
    print(output)
    release = None
    if helm_major_version < 3:
        for rel in yaml.safe_load(output).get("Releases", []):
            if rel.get("Chart") == chart_release_name:
                release = rel
                break
    else:
        for rel in yaml.safe_load(output):
            if rel.get("chart") == chart_release_name:
                release = rel
                break
    assert release, "chart '%s' not found in helm list output:\n%s" % (chart_release_name, output)
    release_name = release.get("Name", release.get("name"))
    assert release_name, "failed to get name for release:\n%s" % yaml.dump(release)
    if helm_major_version < 3:
        return release_name + "-" + chart_name
    return release_name


def install_helm_chart(k8s_cluster, values_path, cont, helm_major_version):
    options = f"--values {values_path} --namespace={k8s_cluster.test_namespace} --debug {CONTAINER_CHART_DIR}"
    if helm_major_version >= 3:
        options = f"--generate-name {options}"
    install_cmd = helm_command_prefix(k8s_cluster, helm_major_version) + f" install {options}"
    print(f"Running Helm install: {install_cmd}")
    output = exec_helm(cont, install_cmd)
    print(f"Helm chart install output:\n{output}")

    daemonset_name = get_daemonset_name(k8s_cluster, cont, helm_major_version)
    print("Waiting for daemonset %s to be ready ..." % daemonset_name)
    try:
        assert wait_for(
            p(daemonset_is_ready, daemonset_name, k8s_cluster.test_namespace), timeout_seconds=120, interval_seconds=2
        ), ("timed out waiting for %s daemonset to be ready!" % daemonset_name)
    finally:
        print(k8s_cluster.exec_kubectl(f"describe daemonset {daemonset_name}", namespace=k8s_cluster.test_namespace))


@contextmanager
def run_helm_image(k8s_cluster, helm_version):
    opts = dict(path=REPO_ROOT_DIR, dockerfile=SCRIPT_DIR / "Dockerfile", buildargs={"VERSION": helm_version})
    with run_service("helm", **opts) as cont:
        output = k8s_cluster.exec_kubectl("config view --raw --flatten")
        copy_file_content_into_container(output, cont, k8s_cluster.kube_config_path)
        yield cont


@pytest.mark.parametrize("helm_version", ["2.15.0", "3.0.0"])
def test_helm(k8s_cluster, helm_version):
    helm_major_version = int(helm_version.split(".")[0])
    with run_helm_image(k8s_cluster, helm_version) as cont:
        with k8s_cluster.create_resources([APP_YAML_PATH]), tiller_rbac_resources(
            k8s_cluster, helm_major_version
        ), fake_backend.start() as backend:
            if helm_major_version < 3:
                init_helm(k8s_cluster, cont, helm_major_version)

            with k8s_cluster.run_tunnels(backend) as proxy_pod_ip:
                with release_values_yaml(k8s_cluster, proxy_pod_ip, backend) as values_path:
                    copy_file_into_container(values_path, cont, values_path)
                    install_helm_chart(k8s_cluster, values_path, cont, helm_major_version)
                    try:
                        assert wait_for(
                            p(
                                has_datapoint,
                                backend,
                                dimensions={"container_name": "prometheus", "application": "helm-test"},
                            ),
                            timeout_seconds=60,
                        )
                        assert wait_for(p(has_datapoint, backend, metric_name="memory.utilization"), timeout_seconds=60)
                    finally:
                        for pod in get_pods_by_labels("app=signalfx-agent", namespace=k8s_cluster.test_namespace):
                            print("pod/%s:" % pod.metadata.name)
                            status = exec_pod_command(
                                pod.metadata.name, AGENT_STATUS_COMMAND, namespace=k8s_cluster.test_namespace
                            )
                            print("Agent Status:\n%s" % status)
                            logs = get_pod_logs(pod.metadata.name, namespace=k8s_cluster.test_namespace)
                            print("Agent Logs:\n%s" % logs)
                        print("\nDatapoints received:")
                        for dp in backend.datapoints:
                            print_dp_or_event(dp)
                        print("\nEvents received:")
                        for event in backend.events:
                            print_dp_or_event(event)
                        print(f"\nDimensions set: {backend.dims}")

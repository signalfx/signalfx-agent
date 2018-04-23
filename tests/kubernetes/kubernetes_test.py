from functools import partial as p
from tests.helpers.util import *
from tests.helpers.assertions import *
from tests.kubernetes.utils import *

import os
import pytest

pytestmark = [pytest.mark.k8s, pytest.mark.kubernetes]

DOCS_DIR = os.environ.get("DOCS_DIR", "/go/src/github.com/signalfx/signalfx-agent/docs/monitors")

# list of tuples for the monitor and respective metrics
# the monitor should be a YAML-based dictionary which will be used for the signalfx-agent agent.yaml configuration
MONITORS_TO_TEST = [
    ({"type": "collectd/cpu"}, get_metrics_from_doc(os.path.join(DOCS_DIR, "collectd-cpu.md"))),
    ({"type": "collectd/interface"}, get_metrics_from_doc(os.path.join(DOCS_DIR, "collectd-interface.md"))),
    ({"type": "collectd/memory"}, get_metrics_from_doc(os.path.join(DOCS_DIR,"collectd-memory.md"))),
    ({"type": "collectd/nginx", "discoveryRule": 'container_image =~ "nginx" && private_port == 80'}, get_metrics_from_doc(os.path.join(DOCS_DIR,"collectd-nginx.md"))),
    ({"type": "collectd/protocols"}, get_metrics_from_doc(os.path.join(DOCS_DIR,"collectd-protocols.md"))),
    ({"type": "collectd/signalfx-metadata", "procFSPath": "/hostfs/proc", "etcPath": "/hostfs/etc", "persistencePath": "/run"}, get_metrics_from_doc(os.path.join(DOCS_DIR,"collectd-signalfx-metadata.md"), ignore=['cpu.utilization_per_core', 'disk.summary_utilization', 'disk.utilization', 'disk_ops.total'])),
    ({"type": "collectd/uptime"}, get_metrics_from_doc(os.path.join(DOCS_DIR,"collectd-uptime.md"))),
    ({"type": "collectd/vmem"}, get_metrics_from_doc(os.path.join(DOCS_DIR,"collectd-vmem.md"))),
    ({"type": "docker-container-stats"}, get_metrics_from_doc(os.path.join(DOCS_DIR,"docker-container-stats.md"), ignore=["memory.stats.swap"])),
    ({"type": "kubelet-stats", "kubeletAPI": {"skipVerify": True, "authType": "serviceAccount"}}, get_metrics_from_doc(os.path.join(DOCS_DIR,"kubelet-stats.md"))),
    ({"type": "kubernetes-cluster", "kubernetesAPI": {"authType": "serviceAccount"}}, get_metrics_from_doc(os.path.join(DOCS_DIR,"kubernetes-cluster.md"))),
    ({"type": "kubernetes-volumes", "kubeletAPI": {"skipVerify": True, "authType": "serviceAccount"}}, ['kubernetes.volume_available_bytes', 'kubernetes.volume_capacity_bytes']),
]

def get_expected_dims(minikube):
    rc, machine_id = minikube.agent.container.exec_run("cat /etc/machine-id")
    if rc != 0:
        machine_id = None
    dims = [
        {"key": "host", "value": minikube.container.attrs['Config']['Hostname']},
        {"key": "kubernetes_cluster", "value": minikube.cluster_name},
        {"key": "kubernetes_namespace", "value": minikube.namespace},
        {"key": "machine_id", "value": machine_id},
        {"key": "metric_source", "value": "kubernetes"}
    ]
    for service in minikube.services:
        if "pod_name" in service["config"].keys():
            pods = get_all_pods_matching_name(service["config"]["pod_name"])
            assert len(pods) > 0, "failed to get pods with name '%s'!" % service["config"]["pod_name"]
            for pod in pods:
                dims.extend([
                    {"key": "container_spec_name", "value": pod.spec.containers[0].name},
                    {"key": "kubernetes_pod_name", "value": pod.metadata.name},
                    {"key": "kubernetes_pod_uid", "value": pod.metadata.uid}
                ])
        if "image" in service["config"].keys():
            containers = minikube.client.containers.list(filters={"ancestor": service["config"]["image"]})
            assert len(containers) > 0, "failed to get containers with ancestor '%s'!" % service["config"]["image"]
            for container in containers:
                dims.extend([
                    {"key": "container_id", "value": container.id},
                    {"key": "container_name", "value": container.name}
                ])
    return dims

def check_for_metrics(backend, metrics, timeout):
    start_time = time.time()
    while True:
        if (time.time() - start_time) > timeout:
            break
        for metric in metrics:
            if has_datapoint_with_metric_name(backend, metric):
                metrics.remove(metric)
        if len(metrics) == 0:
            break
        time.sleep(5)
    return metrics

def check_for_dims(backend, dims, timeout):
    start_time = time.time()
    while True:
        if (time.time() - start_time) > timeout:
            break
        for dim in dims:
            if not dim["value"] or has_datapoint_with_dim(backend, dim["key"], dim["value"]):
                dims.remove(dim)
        if len(dims) == 0:
            break
        time.sleep(5)
    return dims

@pytest.mark.parametrize("monitor,expected_metrics", MONITORS_TO_TEST, ids=[m[0]["type"] for m in MONITORS_TO_TEST])
def test_metrics(monitor, expected_metrics, minikube, k8s_test_timeout):
    if monitor["type"] == 'collectd/nginx' and minikube.agent.observer in ['docker', 'host']:
        pytest.skip("skipping monitor '%s' test for observer '%s'" % (monitor["type"], minikube.agent.observer))
    if len(expected_metrics) == 0:
        pytest.skip("expected metrics is empty; skipping test")
    print("\nCollected %d metric(s) to test for %s." % (len(expected_metrics), monitor["type"]))
    metrics_not_found = check_for_metrics(minikube.agent.backend, expected_metrics, k8s_test_timeout)
    assert len(metrics_not_found) == 0, "timed out waiting for metric(s): %s\n\n%s\n\n" % (metrics_not_found, get_all_logs(minikube))

def test_dims(minikube, k8s_test_timeout):
    expected_dims = get_expected_dims(minikube)
    if len(expected_dims) == 0:
        pytest.skip("expected dimensions is empty; skipping test")
    print("\nCollected %d dimension(s) to test." % len(expected_dims))
    dims_not_found = check_for_dims(minikube.agent.backend, expected_dims, k8s_test_timeout)
    assert len(dims_not_found) == 0, "timed out waiting for dimension(s): %s\n\n%s\n\n" % (dims_not_found, get_all_logs(minikube))


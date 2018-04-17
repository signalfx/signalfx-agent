from functools import partial as p
from tests.helpers.util import *
from tests.helpers.assertions import *
from tests.kubernetes.utils import *

import os
import pytest

pytestmark = [pytest.mark.k8s, pytest.mark.kubernetes]

# list of docs to get metrics from
DOCS_DIR = os.environ.get("DOCS_DIR", "/go/src/github.com/signalfx/signalfx-agent/docs/monitors")
DOCS = [ 
    "collectd-nginx.md",
    "kubelet-stats.md",
    "kubernetes-cluster.md"
]
DOCS = [os.path.join(DOCS_DIR, doc) for doc in DOCS]

def has_all_metrics(backend, metrics):
    for metric in metrics:
        if not has_datapoint_with_metric_name(backend, metric):
            return False
    return True

def has_all_dims(backend, dims):
    for dim in dims:
        if dim["value"] and not has_datapoint_with_dim(backend, dim["key"], dim["value"]):
            return False
    return True

def get_expected_dims(container, cluster_name="minikube", namespace="default", machine_id=None, metric_source="kubernetes", pod_names=[], images=[]):
    client = get_minikube_docker_client(container)
    dims = [
        {"key": "host", "value": container.attrs['Config']['Hostname']},
        {"key": "kubernetes_cluster", "value": cluster_name},
        {"key": "kubernetes_namespace", "value": namespace},
        {"key": "machine_id", "value": machine_id},
        {"key": "metric_source", "value": metric_source}
    ]
    for pod_name in pod_names:
        pods = get_all_pods_with_name(pod_name)
        assert len(pods) > 0, "failed to get pods with name '%s'!" % pod_name
        for pod in pods:
            dims.extend([
                {"key": "container_spec_name", "value": pod.spec.containers[0].name},
                {"key": "kubernetes_pod_name", "value": pod.metadata.name},
                {"key": "kubernetes_pod_uid", "value": pod.metadata.uid}
            ])
    for image in images:
        conts = client.containers.list(filters={"ancestor": image})
        assert len(conts) > 0, "failed to get containers with ancestor '%s'!" % image
        for cont in conts:
            dims.extend([
                {"key": "container_id", "value": cont.id},
                {"key": "container_name", "value": cont.name}
            ])
    return dims

@pytest.fixture
def timeout(request):
    return int(request.config.getoption("--k8s-metrics-timeout"))

def test_dims(request, minikube, backend, timeout):
    #timeout = int(request.config.getoption("--k8s-metrics-timeout"))
    expected_dims = get_expected_dims(minikube, pod_names=['nginx-replication-controller-.*'], images=['nginx:latest'])
    print("\nCollected %d dimensions to test." % len(expected_dims))
    assert wait_for(p(has_all_dims, backend, expected_dims), timeout_seconds=timeout), get_all_logs(minikube)

@pytest.mark.parametrize("monitor", [
    "kubelet-stats",
    "kubernetes-cluster",
    "collectd-nginx"
])
def test_metrics(request, monitor, minikube, backend, timeout):
    #timeout = int(request.config.getoption("--k8s-metrics-timeout"))
    expected_metrics = get_metrics_from_docs(docs=[os.path.join(DOCS_DIR, "%s.md" % monitor)])
    print("\nCollected %d metrics to test from doc(s)." % len(expected_metrics))
    assert wait_for(p(has_all_metrics, backend, expected_metrics), timeout_seconds=timeout), get_all_logs(minikube)


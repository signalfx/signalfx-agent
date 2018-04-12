from functools import partial as p
from tests.helpers.util import *
from tests.helpers.assertions import *
from tests.kubernetes.utils import *

import os
import pytest

# list of docs to get metrics from
DOCS_DIR = os.environ.get("DOCS_DIR", "/go/src/github.com/signalfx/signalfx-agent/docs/monitors")
DOCS = [ 
    "collectd-nginx.md",
    "kubelet-stats.md",
    "kubernetes-cluster.md"
]
DOCS = [os.path.join(DOCS_DIR, i) for i in DOCS]

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

def get_dims(container, client):
    dims = [
        {"key": "host", "value": container.attrs['Config']['Hostname']},
        {"key": "kubernetes_namespace", "value": "default"},
        {"key": "kubernetes_cluster", "value": "minikube"},
        {"key": "machine_id", "value": None},
        {"key": "metric_source", "value": "kubernetes"}
    ]
    for nginx_pod in get_all_pods_with_name('nginx-replication-controller-.*'):
        dims.extend([
            {"key": "container_spec_name", "value": nginx_pod.spec.containers[0].name},
            {"key": "kubernetes_pod_name", "value": nginx_pod.metadata.name},
            {"key": "kubernetes_pod_uid", "value": nginx_pod.metadata.uid}
        ])
    for nginx_container in client.containers.list(filters={"ancestor": "nginx:latest"}):
        dims.extend([
            {"key": "container_id", "value": nginx_container.id},
            {"key": "container_name", "value": nginx_container.name}
        ])
    return dims

@pytest.mark.k8s
@pytest.mark.kubernetes
def test_k8s_metrics(minikube, request):
    metrics_timeout = int(request.config.getoption("--k8s-metrics-timeout"))
    with minikube as [mk_container, mk_docker_client, backend]:
        # test for metrics
        expected_metrics = get_metrics_from_docs(docs=DOCS)
        print("\nCollected %d metrics to test from docs." % len(expected_metrics))
        assert wait_for(p(has_all_metrics, backend, expected_metrics), timeout_seconds=metrics_timeout)
        # test for dimensions
        expected_dims = get_dims(mk_container, mk_docker_client)
        print("\nCollected %d dimensions to test." % len(expected_dims))
        assert wait_for(p(has_all_dims, backend, expected_dims), timeout_seconds=metrics_timeout)


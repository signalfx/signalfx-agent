import os
from functools import partial as p

import pytest

from tests.helpers.assertions import has_datapoint
from tests.helpers.kubernetes.utils import run_k8s_monitors_test, run_k8s_with_agent
from tests.helpers.util import (
    ensure_always,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    wait_for,
)

pytestmark = [pytest.mark.kubernetes_cluster, pytest.mark.monitor_without_endpoints]


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_kubernetes_cluster_in_k8s(agent_image, minikube, k8s_test_timeout, k8s_namespace):
    monitors = [{"type": "kubernetes-cluster", "kubernetesAPI": {"authType": "serviceAccount"}}]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        expected_metrics=get_monitor_metrics_from_selfdescribe(monitors[0]["type"]),
        expected_dims=get_monitor_dims_from_selfdescribe(monitors[0]["type"]),
        test_timeout=k8s_test_timeout,
    )


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_resource_quota_metrics(agent_image, minikube, k8s_namespace):
    monitors = [{"type": "kubernetes-cluster", "kubernetesAPI": {"authType": "serviceAccount"}}]

    with run_k8s_with_agent(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=[os.path.join(os.path.dirname(os.path.realpath(__file__)), "resource_quota.yaml")],
    ) as [backend, _]:

        assert wait_for(
            p(
                has_datapoint,
                backend,
                metric_name="kubernetes.resource_quota_hard",
                dimensions={"quota_name": "object-quota-demo", "resource": "requests.cpu"},
                value=100_000,
            )
        )

        assert wait_for(
            p(
                has_datapoint,
                backend,
                metric_name="kubernetes.resource_quota_hard",
                dimensions={"quota_name": "object-quota-demo", "resource": "persistentvolumeclaims"},
                value=4,
            )
        )

        assert wait_for(
            p(
                has_datapoint,
                backend,
                metric_name="kubernetes.resource_quota_used",
                dimensions={"quota_name": "object-quota-demo", "resource": "persistentvolumeclaims"},
                value=0,
            )
        )

        assert wait_for(
            p(
                has_datapoint,
                backend,
                metric_name="kubernetes.resource_quota_hard",
                dimensions={"quota_name": "object-quota-demo", "resource": "services.loadbalancers"},
                value=2,
            )
        )


def local_file(path):
    return os.path.join(os.path.dirname(os.path.realpath(__file__)), path)


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_kubernetes_cluster_namespace_scope(agent_image, minikube, k8s_namespace):
    monitors = [{"type": "kubernetes-cluster", "kubernetesAPI": {"authType": "serviceAccount"}, "namespace": "good"}]

    with run_k8s_with_agent(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=[local_file("good-pod.yaml"), local_file("bad-pod.yaml")],
    ) as [backend, _]:
        assert wait_for(
            p(has_datapoint, backend, dimensions={"kubernetes_namespace": "good"})
        ), "timed out waiting for good pod metrics"

        assert ensure_always(
            lambda: not has_datapoint(backend, dimensions={"kubernetes_namespace": "bad"})
        ), "got pod metrics from unspecified namespace"

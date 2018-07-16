from functools import partial as p
import os
import pytest

from tests.helpers.assertions import has_datapoint
from tests.helpers.util import (
    get_monitor_metrics_from_selfdescribe,
    get_monitor_dims_from_selfdescribe,
    wait_for
)
from tests.kubernetes.utils import (
    run_k8s_monitors_test,
    run_k8s_with_agent,
)

pytestmark = [pytest.mark.kubernetes_cluster, pytest.mark.monitor_without_endpoints]


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_kubernetes_cluster_in_k8s(agent_image, minikube, k8s_test_timeout, k8s_namespace):
    monitors = [
        {"type": "kubernetes-cluster",
         "kubernetesAPI": {"authType": "serviceAccount"}},
    ]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        expected_metrics=get_monitor_metrics_from_selfdescribe(monitors[0]["type"]),
        expected_dims=get_monitor_dims_from_selfdescribe(monitors[0]["type"]),
        test_timeout=k8s_test_timeout)

@pytest.mark.k8s
@pytest.mark.kubernetes
def test_resource_quota_metrics(agent_image, minikube, k8s_test_timeout, k8s_namespace):
    monitors = [
        {"type": "kubernetes-cluster",
         "kubernetesAPI": {"authType": "serviceAccount"}},
    ]

    with run_k8s_with_agent(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=[os.path.join(os.path.dirname(os.path.realpath(__file__)), "resource_quota.yaml")]) as [backend, _]:

        wait_for(p(has_datapoint, backend,
            metric_name="kubernetes.resource_quota_hard",
            dimensions={"quota_name": "object-quota-demo", "resource": "persistentvolumeclaims"},
            value=4))

        wait_for(p(has_datapoint, backend,
            metric_name="kubernetes.resource_quota_used",
            dimensions={"quota_name": "object-quota-demo", "resource": "persistentvolumeclaims"},
            value=0))

        wait_for(p(has_datapoint, backend,
            metric_name="kubernetes.resource_quota_hard",
            dimensions={"quota_name": "object-quota-demo", "resource": "services.loadbalancers"},
            value=2))

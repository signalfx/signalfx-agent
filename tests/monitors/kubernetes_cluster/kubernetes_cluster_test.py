from functools import partial as p
from pathlib import Path

import pytest

from tests.helpers.assertions import has_datapoint
from tests.helpers.kubernetes.utils import run_k8s_monitors_test
from tests.helpers.util import (
    ensure_always,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    wait_for,
)

pytestmark = [pytest.mark.kubernetes_cluster, pytest.mark.monitor_without_endpoints]

SCRIPT_DIR = Path(__file__).parent.resolve()


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


@pytest.mark.kubernetes
def test_resource_quota_metrics(agent_image, minikube, k8s_namespace):
    yamls = [SCRIPT_DIR / "resource_quota.yaml"]
    with minikube.create_resources(yamls, namespace=k8s_namespace):
        config = """
            monitors:
            - type: kubernetes-cluster
              kubernetesAPI:
                authType: serviceAccount
        """
        with minikube.run_agent(agent_image, config=config, namespace=k8s_namespace) as [_, fake_services]:
            assert wait_for(
                p(
                    has_datapoint,
                    fake_services,
                    metric_name="kubernetes.resource_quota_hard",
                    dimensions={"quota_name": "object-quota-demo", "resource": "requests.cpu"},
                    value=100_000,
                )
            )

            assert wait_for(
                p(
                    has_datapoint,
                    fake_services,
                    metric_name="kubernetes.resource_quota_hard",
                    dimensions={"quota_name": "object-quota-demo", "resource": "persistentvolumeclaims"},
                    value=4,
                )
            )

            assert wait_for(
                p(
                    has_datapoint,
                    fake_services,
                    metric_name="kubernetes.resource_quota_used",
                    dimensions={"quota_name": "object-quota-demo", "resource": "persistentvolumeclaims"},
                    value=0,
                )
            )

            assert wait_for(
                p(
                    has_datapoint,
                    fake_services,
                    metric_name="kubernetes.resource_quota_hard",
                    dimensions={"quota_name": "object-quota-demo", "resource": "services.loadbalancers"},
                    value=2,
                )
            )


@pytest.mark.kubernetes
def test_kubernetes_cluster_namespace_scope(agent_image, minikube, k8s_namespace):
    yamls = [SCRIPT_DIR / "good-pod.yaml", SCRIPT_DIR / "bad-pod.yaml"]
    with minikube.create_resources(yamls, namespace=k8s_namespace):
        config = """
            monitors:
            - type: kubernetes-cluster
              kubernetesAPI:
                authType: serviceAccount
              namespace: good
        """
        with minikube.run_agent(agent_image, config=config, namespace=k8s_namespace) as [_, fake_services]:
            assert wait_for(
                p(has_datapoint, fake_services, dimensions={"kubernetes_namespace": "good"})
            ), "timed out waiting for good pod metrics"
            assert ensure_always(
                lambda: not has_datapoint(fake_services, dimensions={"kubernetes_namespace": "bad"})
            ), "got pod metrics from unspecified namespace"

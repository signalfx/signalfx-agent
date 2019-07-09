from functools import partial as p
from pathlib import Path

import pytest
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import ensure_always, get_monitor_metrics_from_selfdescribe, wait_for
from tests.paths import TEST_SERVICES_DIR

pytestmark = [pytest.mark.kubernetes_cluster, pytest.mark.monitor_without_endpoints]

SCRIPT_DIR = Path(__file__).parent.resolve()


@pytest.mark.kubernetes
def test_kubernetes_cluster_in_k8s(k8s_cluster):
    config = """
    monitors:
     - type: kubernetes-cluster
    """
    yamls = [SCRIPT_DIR / "resource_quota.yaml", TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"]
    with k8s_cluster.create_resources(yamls):
        with k8s_cluster.run_agent(agent_yaml=config) as agent:
            for metric in get_monitor_metrics_from_selfdescribe("kubernetes-cluster"):
                if "replication_controller" in metric:
                    continue
                assert wait_for(p(has_datapoint, agent.fake_services, metric_name=metric))


@pytest.mark.kubernetes
def test_resource_quota_metrics(k8s_cluster):
    yamls = [SCRIPT_DIR / "resource_quota.yaml"]
    with k8s_cluster.create_resources(yamls):
        config = """
            monitors:
            - type: kubernetes-cluster
              kubernetesAPI:
                authType: serviceAccount
        """
        with k8s_cluster.run_agent(agent_yaml=config) as agent:
            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="kubernetes.resource_quota_hard",
                    dimensions={"quota_name": "object-quota-demo", "resource": "requests.cpu"},
                    value=100_000,
                )
            )

            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="kubernetes.resource_quota_hard",
                    dimensions={"quota_name": "object-quota-demo", "resource": "persistentvolumeclaims"},
                    value=4,
                )
            )

            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="kubernetes.resource_quota_used",
                    dimensions={"quota_name": "object-quota-demo", "resource": "persistentvolumeclaims"},
                    value=0,
                )
            )

            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="kubernetes.resource_quota_hard",
                    dimensions={"quota_name": "object-quota-demo", "resource": "services.loadbalancers"},
                    value=2,
                )
            )


@pytest.mark.kubernetes
def test_kubernetes_cluster_namespace_scope(k8s_cluster):
    yamls = [SCRIPT_DIR / "good-pod.yaml", SCRIPT_DIR / "bad-pod.yaml"]
    with k8s_cluster.create_resources(yamls):
        config = """
            monitors:
            - type: kubernetes-cluster
              kubernetesAPI:
                authType: serviceAccount
              namespace: good
        """
        with k8s_cluster.run_agent(agent_yaml=config) as agent:
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"kubernetes_namespace": "good"})
            ), "timed out waiting for good pod metrics"
            assert ensure_always(
                lambda: not has_datapoint(agent.fake_services, dimensions={"kubernetes_namespace": "bad"})
            ), "got pod metrics from unspecified namespace"

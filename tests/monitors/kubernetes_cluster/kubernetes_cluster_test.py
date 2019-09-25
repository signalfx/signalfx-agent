from functools import partial as p
from pathlib import Path
import pytest

from tests.helpers.assertions import any_dim_val_has_prop, has_all_dim_props, has_datapoint
from tests.helpers.util import ensure_always, get_default_monitor_metrics_from_selfdescribe, wait_for
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
            for metric in get_default_monitor_metrics_from_selfdescribe("kubernetes-cluster"):
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


@pytest.mark.kubernetes
def test_stateful_sets(k8s_cluster):
    yamls = [SCRIPT_DIR / "statefulset.yaml"]
    with k8s_cluster.create_resources(yamls) as resources:
        config = """
            monitors:
            - type: kubernetes-cluster
              kubernetesAPI:
                authType: serviceAccount
              extraMetrics:
                - kubernetes.stateful_set.desired
        """
        with k8s_cluster.run_agent(agent_yaml=config) as agent:
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"kubernetes_name": "web"}), timeout_seconds=600
            ), "timed out waiting for statefulset metric"

            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="kubernetes.stateful_set.desired",
                    value=3,
                    dimensions={"kubernetes_name": "web"},
                )
            ), "timed out waiting for statefulset metric"

            assert wait_for(
                p(
                    has_all_dim_props,
                    agent.fake_services,
                    dim_name="kubernetes_uid",
                    dim_value=resources[0].metadata.uid,
                    props={"kubernetes_workload": "StatefulSet"},
                )
            )


@pytest.mark.kubernetes
def test_jobs(k8s_cluster):
    yamls = [SCRIPT_DIR / "job.yaml"]
    with k8s_cluster.create_resources(yamls) as resources:
        config = """
            monitors:
            - type: kubernetes-cluster
              kubernetesAPI:
                authType: serviceAccount
              extraMetrics:
                - kubernetes.job.completions
        """
        with k8s_cluster.run_agent(agent_yaml=config) as agent:
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"kubernetes_name": "pi"}), timeout_seconds=600
            ), f"timed out waiting for job metric"

            assert wait_for(
                p(has_datapoint, agent.fake_services, metric_name="kubernetes.job.completions")
            ), f"timed out waiting for job metric completions"

            assert wait_for(
                p(
                    has_all_dim_props,
                    agent.fake_services,
                    dim_name="kubernetes_uid",
                    dim_value=resources[0].metadata.uid,
                    props={"kubernetes_workload": "Job"},
                ),
                timeout_seconds=300,
            )


@pytest.mark.kubernetes
def test_cronjobs(k8s_cluster):
    yamls = [SCRIPT_DIR / "cronjob.yaml"]
    with k8s_cluster.create_resources(yamls) as resources:
        config = """
            monitors:
            - type: kubernetes-cluster
              kubernetesAPI:
                authType: serviceAccount
              extraMetrics:
                - kubernetes.cronjob.active
        """
        with k8s_cluster.run_agent(agent_yaml=config) as agent:
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"kubernetes_name": "pi-cron"}), timeout_seconds=600
            ), "timed out waiting for cronjob metric"

            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="kubernetes.cronjob.active",
                    dimensions={"kubernetes_name": "pi-cron"},
                )
            ), "timed out waiting for cronjob metric 'kubernetes.cronjob.active'"

            assert wait_for(
                p(
                    has_all_dim_props,
                    agent.fake_services,
                    dim_name="kubernetes_uid",
                    dim_value=resources[0].metadata.uid,
                    props={"kubernetes_workload": "CronJob"},
                ),
                timeout_seconds=300,
            )


@pytest.mark.kubernetes
def test_pods(k8s_cluster):
    config = """
    monitors:
     - type: kubernetes-cluster
    """
    yamls = [SCRIPT_DIR / "resource_quota.yaml", TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"]
    with k8s_cluster.create_resources(yamls):
        with k8s_cluster.run_agent(agent_yaml=config) as agent:
            assert wait_for(
                p(
                    any_dim_val_has_prop,
                    agent.fake_services,
                    dim_name="kubernetes_pod_uid",
                    prop_name="pod_creation_timestamp",
                )
            )


@pytest.mark.kubernetes
def test_persistent_volume_claims(k8s_cluster):
    config = """
    monitors:
     - type: kubernetes-cluster
       extraMetrics:
        - kubernetes.persistent_volume_claim_phase
    """
    yamls = [SCRIPT_DIR / "persistent_volume_claim.yaml"]
    with k8s_cluster.create_resources(yamls):
        with k8s_cluster.run_agent(agent_yaml=config) as agent:
            assert wait_for(
                p(
                    any_dim_val_has_prop,
                    agent.fake_services,
                    dim_name="kubernetes_uid",
                    prop_name="persistent_volume_claim_creation_timestamp",
                )
            )

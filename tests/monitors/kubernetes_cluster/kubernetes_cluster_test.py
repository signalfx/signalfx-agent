from functools import partial as p
from pathlib import Path

import pytest
from kubernetes import client as k8s_client
from tests.helpers.assertions import has_all_dim_props, has_datapoint, has_no_datapoint
from tests.helpers.util import (
    ensure_always,
    get_default_monitor_metrics_from_selfdescribe,
    get_stripped_container_id,
    wait_for,
    wait_for_assertion,
)
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
                    props={
                        "statefulset_creation_timestamp": resources[0].metadata.creation_timestamp.strftime(
                            "%Y-%m-%dT%H:%M:%SZ"
                        ),
                        "kubernetes_workload": "StatefulSet",
                        "kubernetes_workload_name": resources[0].metadata.name,
                    },
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
                    props={
                        "job_creation_timestamp": resources[0].metadata.creation_timestamp.strftime(
                            "%Y-%m-%dT%H:%M:%SZ"
                        ),
                        "kubernetes_workload": "Job",
                        "kubernetes_workload_name": resources[0].metadata.name,
                    },
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
                    props={
                        "cronjob_creation_timestamp": resources[0].metadata.creation_timestamp.strftime(
                            "%Y-%m-%dT%H:%M:%SZ"
                        ),
                        "kubernetes_workload": "CronJob",
                        "kubernetes_workload_name": resources[0].metadata.name,
                    },
                ),
                timeout_seconds=300,
            )


@pytest.mark.kubernetes
def test_pods(k8s_cluster):
    config = """
    monitors:
     - type: kubernetes-cluster
    """
    yamls = [TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"]
    with k8s_cluster.create_resources(yamls) as resources:

        with k8s_cluster.run_agent(agent_yaml=config) as agent:
            pods = k8s_client.CoreV1Api().list_namespaced_pod(
                k8s_cluster.test_namespace, watch=False, label_selector="app=nginx"
            )
            for pod in pods.items:
                assert wait_for(
                    p(
                        has_all_dim_props,
                        agent.fake_services,
                        dim_name="kubernetes_pod_uid",
                        dim_value=pod.metadata.uid,
                        props={
                            "pod_creation_timestamp": pod.metadata.creation_timestamp.strftime("%Y-%m-%dT%H:%M:%SZ"),
                            "deployment": resources[1].metadata.name,
                            "kubernetes_workload_name": pod.metadata.owner_references[0].name,
                            "kubernetes_workload": pod.metadata.owner_references[0].kind,
                        },
                    )
                )


@pytest.mark.kubernetes
def test_deployments(k8s_cluster):
    config = """
    monitors:
     - type: kubernetes-cluster
    """
    yamls = [TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"]
    with k8s_cluster.create_resources(yamls) as resources:
        with k8s_cluster.run_agent(agent_yaml=config) as agent:
            assert wait_for(
                p(
                    has_all_dim_props,
                    agent.fake_services,
                    dim_name="kubernetes_uid",
                    dim_value=resources[1].metadata.uid,
                    props={
                        "deployment": resources[1].metadata.name,
                        "kubernetes_workload": "Deployment",
                        "kubernetes_workload_name": resources[1].metadata.name,
                        "deployment_creation_timestamp": resources[1].metadata.creation_timestamp.strftime(
                            "%Y-%m-%dT%H:%M:%SZ"
                        ),
                    },
                )
            )


CONTAINER_RESOURCE_METRICS = {
    "request": {
        "cpu": "kubernetes.container_cpu_request",
        "memory": "kubernetes.container_memory_request",
        "ephemeral-storage": "kubernetes.container_ephemeral_storage_request",
    },
    "limit": {
        "cpu": "kubernetes.container_cpu_limit",
        "memory": "kubernetes.container_memory_limit",
        "ephemeral-storage": "kubernetes.container_ephemeral_storage_limit",
    },
}


def get_current_container_status(container_state):
    if container_state.running:
        return "running"
    if container_state.terminated:
        return "terminated"
    if container_state.waiting:
        return "waiting"

    return None


@pytest.mark.kubernetes
def test_containers(k8s_cluster):
    config = f"""
    monitors:
     - type: kubernetes-cluster
       extraMetrics:
        - {CONTAINER_RESOURCE_METRICS["request"]["cpu"]}
        - {CONTAINER_RESOURCE_METRICS["request"]["memory"]}
        - {CONTAINER_RESOURCE_METRICS["limit"]["cpu"]}
        - {CONTAINER_RESOURCE_METRICS["limit"]["memory"]}
    """
    yamls = [TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"]
    with k8s_cluster.create_resources(yamls):
        pods = k8s_client.CoreV1Api().list_namespaced_pod(
            k8s_cluster.test_namespace, watch=False, label_selector="app=nginx"
        )
        with k8s_cluster.run_agent(agent_yaml=config) as agent:
            for pod in pods.items:
                # Pod spec does not have knowledge about container ids
                # This dict maintains a map between container name (which)
                # is available in Pod spec and the respective container id
                containers_cache = {}
                for container_status in pod.status.container_statuses:
                    containers_cache[container_status.name] = get_stripped_container_id(container_status.container_id)
                    current_status = get_current_container_status(container_status.state)

                    if containers_cache[container_status.name] == "":
                        continue

                    assert wait_for(
                        p(
                            has_datapoint,
                            agent.fake_services,
                            dimensions={"container_id": containers_cache[container_status.name]},
                        )
                    )
                    assert wait_for(
                        p(
                            has_all_dim_props,
                            agent.fake_services,
                            dim_name="container_id",
                            dim_value=containers_cache[container_status.name],
                            props={"container_status": current_status},
                        )
                    )

                # Check for optional container resource metrics
                for container in pod.spec.containers:
                    valid_metrics = {
                        "request": {"cpu": False, "memory": False, "ephemeral-storage": False},
                        "limit": {"cpu": False, "memory": False, "ephemeral-storage": False},
                    }
                    if container.resources:
                        if container.resources.requests:
                            if container.resources.requests.get("cpu", None):
                                valid_metrics["request"]["cpu"] = True
                            if container.resources.requests.get("memory", None):
                                valid_metrics["request"]["memory"] = True
                            if container.resources.requests.get("ephemeral-storage", None):
                                valid_metrics["request"]["ephemeral-storage"] = True
                        if container.resources.limits:
                            if container.resources.limits.get("cpu", None):
                                valid_metrics["limit"]["cpu"] = True
                            if container.resources.limits.get("memory", None):
                                valid_metrics["limit"]["memory"] = True
                            if container.resources.limits.get("ephemeral-storage", None):
                                valid_metrics["limit"]["ephemeral-storage"] = True

                    for group, resource in CONTAINER_RESOURCE_METRICS.items():
                        for resource_type, metric in resource.items():
                            if valid_metrics[group][resource_type]:
                                assert wait_for(
                                    p(
                                        has_datapoint,
                                        agent.fake_services,
                                        metric_name=metric,
                                        dimensions={"container_id": containers_cache[container.name]},
                                    )
                                )
                            else:
                                assert wait_for(
                                    p(
                                        has_no_datapoint,
                                        agent.fake_services,
                                        metric_name=metric,
                                        dimensions={"container_id": containers_cache[container.name]},
                                    )
                                )


@pytest.mark.kubernetes
def test_node_metrics_and_props(k8s_cluster):
    config = """
            monitors:
            - type: kubernetes-cluster
        """
    with k8s_cluster.run_agent(agent_yaml=config) as agent:
        for node in k8s_cluster.client.CoreV1Api().list_node().items:
            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="kubernetes.node_ready",
                    dimensions={"kubernetes_node": node.metadata.name, "kubernetes_node_uid": node.metadata.uid},
                ),
                timeout_seconds=100,
            ), "timed out waiting for node ready metric"

            expected_props = {k: v for k, v in node.metadata.labels.items() if len(v) > 0}
            expected_props["kubernetes_node"] = node.metadata.name

            def has_props(node, props):
                assert (
                    {k.replace("/", "_").replace(".", "_"): v for k, v in props.items()}.items()
                    <= agent.fake_services.dims["kubernetes_node_uid"]
                    .get(node.metadata.uid, {})
                    .get("customProperties", {})
                    .items()
                )

            wait_for_assertion(p(has_props, node, expected_props))

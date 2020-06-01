from functools import partial as p

import pytest
from tests.helpers.assertions import has_datapoint, has_no_datapoint
from tests.helpers.metadata import Metadata
from tests.helpers.util import ensure_always, wait_for

pytestmark = [pytest.mark.kubelet_stats, pytest.mark.monitor_without_endpoints]

# A reliably present custom metric name
CUSTOM_METRIC = "container_start_time_seconds"
CUSTOM_METRIC_POD_METRIC = "pod_ephemeral_storage_capacity_bytes"
METADATA = Metadata.from_package("cadvisor", mon_type="kubelet-stats")


def _skip_if_1_18_or_newer(k8s_cluster):
    print(k8s_cluster.get_cluster_version())
    if tuple([int(v) for v in k8s_cluster.get_cluster_version().lstrip("v").split("-")[0].split(".")]) >= (1, 18, 0):

        pytest.skip("skipping since cluster is newer than 1.18")


@pytest.mark.kubernetes
def test_kubelet_stats_defaults(k8s_cluster):
    _skip_if_1_18_or_newer(k8s_cluster)

    config = """
     monitors:
      - type: kubelet-stats
        kubeletAPI:
          skipVerify: true
          authType: serviceAccount
     """
    with k8s_cluster.run_agent(agent_yaml=config) as agent:
        if "docker" in k8s_cluster.container_runtimes:
            assert wait_for(
                p(has_datapoint, agent.fake_services, metric_name="pod_network_receive_bytes_total")
            ), "Didn't get network datapoint"
        assert wait_for(
            p(has_datapoint, agent.fake_services, metric_name="container_cpu_utilization")
        ), "Didn't get cpu datapoint"
        assert wait_for(
            p(has_datapoint, agent.fake_services, metric_name="container_memory_usage_bytes")
        ), "Didn't get memory datapoint"
        assert ensure_always(p(has_no_datapoint, agent.fake_services, metric_name=CUSTOM_METRIC), timeout_seconds=5)


@pytest.mark.kubernetes
def test_kubelet_stats_extra(k8s_cluster):
    _skip_if_1_18_or_newer(k8s_cluster)

    config = f"""
     monitors:
      - type: kubelet-stats
        kubeletAPI:
          skipVerify: true
          authType: serviceAccount
        extraMetrics:
         - {CUSTOM_METRIC}
     """
    with k8s_cluster.run_agent(agent_yaml=config) as agent:
        assert wait_for(p(has_datapoint, agent.fake_services, metric_name=CUSTOM_METRIC))


@pytest.mark.kubernetes
def test_kubelet_stats_extra_pod_metric(k8s_cluster):
    _skip_if_1_18_or_newer(k8s_cluster)

    config = f"""
     monitors:
      - type: kubelet-stats
        kubeletAPI:
          skipVerify: true
          authType: serviceAccount
        extraMetrics:
         - {CUSTOM_METRIC_POD_METRIC}
     """
    with k8s_cluster.run_agent(agent_yaml=config) as agent:
        assert wait_for(p(has_datapoint, agent.fake_services, metric_name=CUSTOM_METRIC_POD_METRIC))


@pytest.mark.kubernetes
def test_kubelet_stats_extra_pod_metric_group(k8s_cluster):
    _skip_if_1_18_or_newer(k8s_cluster)

    config = f"""
     monitors:
      - type: kubelet-stats
        kubeletAPI:
          skipVerify: true
          authType: serviceAccount
        extraGroups: [podEphemeralStats]
     """
    with k8s_cluster.run_agent(agent_yaml=config) as agent:
        for metric in METADATA.metrics_by_group.get("podEphemeralStats", []):
            assert wait_for(p(has_datapoint, agent.fake_services, metric_name=metric), timeout_seconds=100)

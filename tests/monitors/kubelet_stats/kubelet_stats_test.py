from functools import partial as p

import pytest
from tests.helpers.assertions import has_datapoint, has_no_datapoint
from tests.helpers.util import ensure_always, wait_for

pytestmark = [pytest.mark.kubelet_stats, pytest.mark.monitor_without_endpoints]


# A reliably present custom metric name
CUSTOM_METRIC = "container_start_time_seconds"


@pytest.mark.kubernetes
def test_kubelet_stats_defaults(k8s_cluster):
    config = """
     monitors:
      - type: kubelet-stats
        kubeletAPI:
          skipVerify: true
          authType: serviceAccount
     """
    with k8s_cluster.run_agent(agent_yaml=config) as agent:
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

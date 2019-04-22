from functools import partial as p

import pytest
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import wait_for

pytestmark = [pytest.mark.kubelet_stats, pytest.mark.monitor_without_endpoints]


@pytest.mark.kubernetes
def test_kubelet_stats(k8s_cluster):
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
            p(has_datapoint, agent.fake_services, metric_name="container_cpu_user_seconds_total")
        ), "Didn't get cpu datapoint"
        assert wait_for(
            p(has_datapoint, agent.fake_services, metric_name="container_memory_usage_bytes")
        ), "Didn't get memory datapoint"

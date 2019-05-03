from functools import partial as p

import pytest
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import wait_for

pytestmark = [pytest.mark.kubernetes_volumes, pytest.mark.monitor_without_endpoints]


@pytest.mark.kubernetes
def test_kubernetes_volumes_in_k8s(k8s_cluster):
    config = """
      monitors:
       - type: kubernetes-volumes
         kubeletAPI:
           skipVerify: true
           authType: serviceAccount
      """
    with k8s_cluster.run_agent(config) as agent:
        assert wait_for(p(has_datapoint, agent.fake_services, metric_name="kubernetes.volume_available_bytes"))
        assert wait_for(p(has_datapoint, agent.fake_services, metric_name="kubernetes.volume_capacity_bytes"))

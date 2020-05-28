import pytest
from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify

METADATA = Metadata.from_package("kubernetes/kubeletmetrics")


@pytest.mark.kubernetes
def test_kubelet_all_metrics(k8s_cluster):
    config = """
     monitors:
      - type: kubelet-metrics
        kubeletAPI:
          skipVerify: true
          authType: serviceAccount
        extraMetrics: ['*']
     """
    with k8s_cluster.run_agent(agent_yaml=config) as agent:
        verify(agent, METADATA.all_metrics)


@pytest.mark.kubernetes
def test_kubelet_use_pod_endpoints(k8s_cluster):
    config = """
     monitors:
      - type: kubelet-metrics
        usePodsEndpoint: true
        kubeletAPI:
          skipVerify: true
          authType: serviceAccount
        extraMetrics: ['*']
     """
    with k8s_cluster.run_agent(agent_yaml=config) as agent:
        verify(agent, METADATA.all_metrics)

import pytest
from tests.helpers.kubernetes import LATEST_ONLY
from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify

pytestmark = [pytest.mark.kubernetes]
METADATA = Metadata.from_package("kubernetes/scheduler")


@pytest.mark.kubernetes
@LATEST_ONLY
def test_kubernetes_scheduler(k8s_cluster):
    config = """
        observers:
        - type: k8s-api

        monitors:
        - type: kubernetes-scheduler
          discoveryRule: kubernetes_pod_name =~ "kube-scheduler"
          port: 10251
          extraMetrics: ["*"]
     """
    with k8s_cluster.run_agent(config) as agent:
        verify(agent, METADATA.all_metrics)

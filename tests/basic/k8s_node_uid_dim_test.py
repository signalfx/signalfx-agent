from functools import partial as p

import pytest
from tests.helpers.assertions import has_datapoint, has_datapoint_with_dim_key
from tests.helpers.util import ensure_always, wait_for
from tests.paths import TEST_SERVICES_DIR


@pytest.mark.kubernetes
def test_node_uid_host_dim(k8s_cluster):
    config = """
    monitors:
     - type: kubelet-stats
     - type: cpu
    """
    yamls = [TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"]
    with k8s_cluster.create_resources(yamls):
        with k8s_cluster.run_agent(agent_yaml=config) as agent:
            for node in k8s_cluster.client.CoreV1Api().list_node().items:
                assert wait_for(
                    p(has_datapoint, agent.fake_services, dimensions={"kubernetes_node_uid": node.metadata.uid})
                )


@pytest.mark.kubernetes
def test_node_uid_host_dim_kubernetes_cluster_config(k8s_cluster):
    config = """
    monitors:
     - type: kubelet-stats
     - type: cpu
     - type: kubernetes-cluster
       # This will make it fail
       kubernetesAPI:
         authType: none
    """
    yamls = [TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"]
    with k8s_cluster.create_resources(yamls):
        with k8s_cluster.run_agent(agent_yaml=config, wait_for_ready=False) as agent:
            # If it works for one node it should work for all of them
            node = k8s_cluster.client.CoreV1Api().list_node().items[0]

            def no_node_uid_dim():
                return not has_datapoint(agent.fake_services, dimensions={"kubernetes_node_uid": node.metadata.uid})

            def no_host_dim():
                return not has_datapoint_with_dim_key(agent.fake_services, "host")

            assert ensure_always(no_node_uid_dim)
            assert ensure_always(no_host_dim), "no metrics should come through if cannot get node uid"

            # We should get this error since we aren't using in-cluster auth.
            assert "certificate signed by unknown authority" in agent.get_logs()

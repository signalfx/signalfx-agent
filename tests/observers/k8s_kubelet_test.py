from functools import partial as p

import pytest
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import wait_for
from tests.paths import TEST_SERVICES_DIR


@pytest.mark.kubernetes
def test_k8s_kubelet_observer_basic(k8s_cluster):
    nginx_yaml = TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"
    with k8s_cluster.create_resources([nginx_yaml]):
        config = f"""
            observers:
             - type: k8s-kubelet
               kubeletAPI:
                 authType: serviceAccount
            monitors:
             - type: collectd/nginx
               discoveryRule: 'discovered_by == "k8s-kubelet" && kubernetes_namespace == "{k8s_cluster.test_namespace}" && port == 80 && container_spec_name == "nginx"'
               url: "http://{{{{.Host}}}}:{{{{.Port}}}}/nginx_status"
        """
        with k8s_cluster.run_agent(config) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"plugin": "nginx"}))

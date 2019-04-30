from functools import partial as p

import pytest
import yaml
from tests.helpers.assertions import has_datapoint
from tests.helpers.kubernetes.utils import add_pod_spec_annotations, get_discovery_rule
from tests.helpers.util import wait_for
from tests.paths import TEST_SERVICES_DIR


@pytest.mark.kubernetes
def test_k8s_api_observer_basic(k8s_cluster):
    nginx_yaml = TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"
    discovery_rule = get_discovery_rule(nginx_yaml, "k8s-api", namespace=k8s_cluster.test_namespace)
    with k8s_cluster.create_resources([nginx_yaml]):
        config = f"""
            observers:
             - type: k8s-api
            monitors:
             - type: collectd/nginx
               discoveryRule: '{discovery_rule}'
               url: "http://{{{{.Host}}}}:{{{{.Port}}}}/nginx_status"
        """
        with k8s_cluster.run_agent(config) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"plugin": "nginx"}))


@pytest.mark.kubernetes
def test_config_from_annotations(k8s_cluster):
    nginx_yaml = TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"
    nginx_with_annotations = yaml.dump_all(
        add_pod_spec_annotations(
            yaml.safe_load_all(nginx_yaml.read_bytes()),
            {
                "agent.signalfx.com/monitorType.http": "collectd/nginx",
                "agent.signalfx.com/config.80.extraDimensions": "{source: myapp}",
            },
        )
    )
    with k8s_cluster.create_resources([nginx_with_annotations]):
        config = f"""
            observers:
             - type: k8s-api
            monitors: []
        """
        with k8s_cluster.run_agent(config) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"plugin": "nginx", "source": "myapp"}))

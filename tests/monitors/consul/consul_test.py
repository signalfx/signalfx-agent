import string
from functools import partial as p
from pathlib import Path

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_metric_name, tcp_socket_open
from tests.helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test
from tests.helpers.util import (
    container_ip,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_container,
    wait_for,
)

pytestmark = [pytest.mark.collectd, pytest.mark.consul, pytest.mark.monitor_with_endpoints]

DIR = Path(__file__).parent.resolve()
CONSUL_CONFIG = string.Template(
    """
monitors:
  - type: collectd/consul
    host: $host
    port: 8500
    enhancedMetrics: true
"""
)


def test_consul():
    with run_container("consul:0.9.3") as consul_cont:
        host = container_ip(consul_cont)
        config = CONSUL_CONFIG.substitute(host=host)

        assert wait_for(p(tcp_socket_open, host, 8500), 60), "consul service didn't start"

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_metric_name, agent.fake_services, "gauge.consul.catalog.services.total"), 60
            ), "Didn't get consul datapoints"


@pytest.mark.kubernetes
def test_consul_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = DIR / "consul-k8s.yaml"
    monitors = [
        {
            "type": "collectd/consul",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
            "aclToken": "testing123",
            "signalFxAccessToken": "testing123",
            "enhancedMetrics": True,
        }
    ]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=[yaml],
        observer=k8s_observer,
        expected_metrics=get_monitor_metrics_from_selfdescribe(monitors[0]["type"]),
        expected_dims=get_monitor_dims_from_selfdescribe(monitors[0]["type"]),
        test_timeout=k8s_test_timeout,
    )

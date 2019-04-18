import string
from functools import partial as p
from pathlib import Path

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test
from tests.helpers.util import (
    container_ip,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_service,
    wait_for,
)

pytestmark = [pytest.mark.collectd, pytest.mark.haproxy, pytest.mark.monitor_with_endpoints]

DIR = Path(__file__).parent.resolve()
MONITOR_CONFIG = string.Template(
    """
monitors:
- type: collectd/haproxy
  host: $host
  port: 9000
  enhancedMetrics: true
"""
)


@pytest.mark.parametrize("version", ["latest"])
def test_haproxy(version):
    with run_service("haproxy", buildargs={"HAPROXY_VERSION": version}) as service_container:
        host = container_ip(service_container)
        config = MONITOR_CONFIG.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 9000), 120), "haproxy not listening on port"
        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "haproxy")
            ), "didn't get datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_haproxy_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = DIR / "haproxy-k8s.yaml"
    monitors = [
        {
            "type": "collectd/haproxy",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
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

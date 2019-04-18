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
    run_service,
    wait_for,
)

pytestmark = [pytest.mark.collectd, pytest.mark.cassandra, pytest.mark.monitor_with_endpoints]


DIR = Path(__file__).parent.resolve()
CASSANDRA_CONFIG = string.Template(
    """
monitors:
  - type: collectd/cassandra
    host: $host
    port: 7199
    username: cassandra
    password: cassandra
"""
)


@pytest.mark.flaky(reruns=2)
def test_cassandra():
    with run_service("cassandra") as cassandra_cont:
        host = container_ip(cassandra_cont)
        config = CASSANDRA_CONFIG.substitute(host=host)

        # Wait for the JMX port to be open in the container
        assert wait_for(p(tcp_socket_open, host, 7199), 60), "Cassandra JMX didn't start"

        with Agent.run(config) as agent:
            assert wait_for(
                p(
                    has_datapoint_with_metric_name,
                    agent.fake_services,
                    "counter.cassandra.ClientRequest.Read.Latency.Count",
                ),
                60,
            ), "Didn't get Cassandra datapoints"


@pytest.mark.kubernetes
def test_cassandra_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = DIR / "cassandra-k8s.yaml"
    monitors = [
        {
            "type": "collectd/cassandra",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
            "username": "testuser",
            "password": "testing123",
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

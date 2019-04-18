from functools import partial as p
from pathlib import Path
from textwrap import dedent

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test
from tests.helpers.util import (
    container_ip,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_container,
    wait_for,
)

pytestmark = [pytest.mark.collectd, pytest.mark.rabbitmq, pytest.mark.monitor_with_endpoints]

DIR = Path(__file__).parent.resolve()


def test_rabbitmq():
    with run_container("rabbitmq:3.6-management") as rabbitmq_cont:
        host = container_ip(rabbitmq_cont)
        config = dedent(
            f"""
            monitors:
              - type: collectd/rabbitmq
                host: {host}
                port: 15672
                username: guest
                password: guest
                collectNodes: true
                collectChannels: true
            """
        )

        assert wait_for(p(tcp_socket_open, host, 15672), 60), "service didn't start"

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "rabbitmq")
            ), "Didn't get rabbitmq datapoints"
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin_instance", "%s-15672" % host)
            ), "Didn't get expected plugin_instance dimension"


def test_rabbitmq_broker_name():
    with run_container("rabbitmq:3.6-management") as rabbitmq_cont:
        host = container_ip(rabbitmq_cont)
        assert wait_for(p(tcp_socket_open, host, 15672), 60), "service didn't start"

        with Agent.run(
            dedent(
                f"""
            monitors:
              - type: collectd/rabbitmq
                host: {host}
                brokerName: '{{{{.host}}}}-{{{{.username}}}}'
                port: 15672
                username: guest
                password: guest
                collectNodes: true
                collectChannels: true
            """
            )
        ) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin_instance", "%s-guest" % host)
            ), "Didn't get expected plugin_instance dimension"


@pytest.mark.kubernetes
def test_rabbitmq_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = DIR / "rabbitmq-k8s.yaml"
    monitors = [
        {
            "type": "collectd/rabbitmq",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
            "collectChannels": True,
            "collectConnections": True,
            "collectExchanges": True,
            "collectNodes": True,
            "collectQueues": True,
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

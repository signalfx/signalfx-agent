"""
Tests for the collectd/activemq monitor
"""
from functools import partial as p
from pathlib import Path
from textwrap import dedent

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import any_metric_found, tcp_socket_open
from tests.helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test
from tests.helpers.util import (
    container_ip,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_service,
    wait_for,
)

pytestmark = [pytest.mark.collectd, pytest.mark.activemq, pytest.mark.monitor_with_endpoints]

SCRIPT_DIR = Path(__file__).parent.resolve()


def test_activemq():
    with run_service("activemq") as activemq_container:
        host = container_ip(activemq_container)
        config = dedent(
            f"""
            monitors:
              - type: collectd/activemq
                host: {host}
                port: 1099
                serviceURL: service:jmx:rmi:///jndi/rmi://{host}:1099/jmxrmi
                username: testuser
                password: testing123
        """
        )
        assert wait_for(p(tcp_socket_open, host, 1099), 60), "service didn't start"
        with Agent.run(config) as agent:
            metrics = get_monitor_metrics_from_selfdescribe("collectd/activemq")
            assert wait_for(p(any_metric_found, agent.fake_services, metrics)), "Didn't get activemq datapoints"


@pytest.mark.kubernetes
def test_activemq_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = SCRIPT_DIR / "activemq-k8s.yaml"
    build_opts = {"tag": "activemq:k8s-test"}
    minikube.build_image("activemq", build_opts)
    monitors = [
        {
            "type": "collectd/activemq",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
            "serviceURL": "service:jmx:rmi:///jndi/rmi://{{.Host}}:{{.Port}}/jmxrmi",
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

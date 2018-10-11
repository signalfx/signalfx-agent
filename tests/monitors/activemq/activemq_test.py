"""
Tests for the collectd/activemq monitor
"""
import os
from functools import partial as p
from textwrap import dedent

import pytest

from helpers.assertions import any_metric_found, tcp_socket_open
from helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test
from helpers.util import (
    container_ip,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_agent,
    run_service,
    wait_for,
)

pytestmark = [pytest.mark.collectd, pytest.mark.activemq, pytest.mark.monitor_with_endpoints]


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
        with run_agent(config) as [backend, _, _]:
            metrics = get_monitor_metrics_from_selfdescribe("collectd/activemq")
            assert wait_for(p(any_metric_found, backend, metrics)), "Didn't get activemq datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_activemq_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = os.path.join(os.path.dirname(os.path.realpath(__file__)), "activemq-k8s.yaml")
    dockerfile_dir = os.path.join(os.path.dirname(os.path.realpath(__file__)), "../../../test-services/activemq")
    build_opts = {"tag": "activemq:k8s-test"}
    minikube.build_image(dockerfile_dir, build_opts)
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

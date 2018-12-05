"""
Tests for the collectd/apache monitor
"""
import os
import string
from functools import partial as p

import pytest

from helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test
from helpers.util import (
    container_ip,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_agent,
    run_service,
    wait_for,
)

pytestmark = [pytest.mark.collectd, pytest.mark.apache, pytest.mark.monitor_with_endpoints]

APACHE_CONFIG = string.Template(
    """
monitors:
  - type: collectd/apache
    host: $host
    port: 80
"""
)


def test_apache():
    with run_service("apache") as apache_container:
        host = container_ip(apache_container)
        config = APACHE_CONFIG.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 80), 60), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "apache")), "Didn't get apache datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_apache_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = os.path.join(os.path.dirname(os.path.realpath(__file__)), "apache-k8s.yaml")
    monitors = [
        {
            "type": "collectd/apache",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
            "url": "http://{{.Host}}:{{.Port}}/mod_status?auto",
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

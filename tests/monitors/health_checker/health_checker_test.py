from functools import partial as p
from textwrap import dedent
import os
import pytest
import string

from tests.helpers.util import wait_for, run_agent, run_service, container_ip
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.kubernetes.utils import (
    run_k8s_monitors_test,
    get_discovery_rule,
)

pytestmark = [pytest.mark.collectd, pytest.mark.health_checker, pytest.mark.monitor_with_endpoints]

config = string.Template("""
monitors:
  - type: collectd/health-checker
    host: $host
    port: 80
    tcpCheck: true
""")


def test_health_checker_tcp():
    with run_service("nginx") as nginx_container:
        host = container_ip(nginx_container)
        assert wait_for(p(tcp_socket_open, host, 80), 60), "service didn't start"

        with run_agent(config.substitute(host=host)) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "health_checker")), \
                "Didn't get health_checker datapoints"


def test_health_checker_http():
    with run_service("nginx") as nginx_container:
        host = container_ip(nginx_container)
        assert wait_for(p(tcp_socket_open, host, 80), 60), "service didn't start"

        with run_agent(string.Template(dedent("""
        monitors:
          - type: collectd/health-checker
            host: $host
            port: 80
            path: /nonexistent
        """)).substitute(host=host)) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "health_checker")), \
                "Didn't get health_checker datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_health_checker_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = os.path.join(os.path.dirname(os.path.realpath(__file__)), "health-checker-k8s.yaml")
    monitors = [
        {"type": "collectd/health-checker",
         "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
         "url": 'http://{{.Host}}:{{.Port}}/health',
         "jsonKey": "status",
         "jsonVal": "ok"},
    ]
    with open(os.path.join(os.path.dirname(os.path.realpath(__file__)), "metrics.txt"), "r") as fd:
        expected_metrics = {m.strip() for m in fd.readlines() if len(m.strip()) > 0}
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=[yaml],
        observer=k8s_observer,
        expected_metrics=expected_metrics,
        test_timeout=k8s_test_timeout)


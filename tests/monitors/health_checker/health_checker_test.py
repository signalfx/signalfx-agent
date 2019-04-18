import string
from functools import partial as p
from pathlib import Path
from textwrap import dedent

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test, get_metrics
from tests.helpers.util import container_ip, run_service, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.health_checker, pytest.mark.monitor_with_endpoints]

CONFIG = string.Template(
    """
monitors:
  - type: collectd/health-checker
    host: $host
    port: 80
    tcpCheck: true
"""
)

SCRIPT_DIR = Path(__file__).parent.resolve()


def test_health_checker_tcp():
    with run_service("nginx") as nginx_container:
        host = container_ip(nginx_container)
        assert wait_for(p(tcp_socket_open, host, 80), 60), "service didn't start"

        with Agent.run(CONFIG.substitute(host=host)) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "health_checker")
            ), "Didn't get health_checker datapoints"


def test_health_checker_http():
    with run_service("nginx") as nginx_container:
        host = container_ip(nginx_container)
        assert wait_for(p(tcp_socket_open, host, 80), 60), "service didn't start"

        with Agent.run(
            string.Template(
                dedent(
                    """
        monitors:
          - type: collectd/health-checker
            host: $host
            port: 80
            path: /nonexistent
        """
                )
            ).substitute(host=host)
        ) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "health_checker")
            ), "Didn't get health_checker datapoints"


@pytest.mark.windows
def test_health_checker_http_windows():
    with Agent.run(
        string.Template(
            dedent(
                """
    monitors:
      - type: collectd/health-checker
        host: $host
        port: 80
        path: /
    """
            )
        ).substitute(host="localhost")
    ) as agent:
        assert wait_for(
            p(has_datapoint_with_dim, agent.fake_services, "plugin", "health_checker")
        ), "Didn't get health_checker datapoints"


@pytest.mark.kubernetes
def test_health_checker_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = SCRIPT_DIR / "health-checker-k8s.yaml"
    monitors = [
        {
            "type": "collectd/health-checker",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
            "path": "/health",
            "jsonKey": "status",
            "jsonVal": "ok",
        }
    ]
    expected_metrics = get_metrics(SCRIPT_DIR)
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=[yaml],
        observer=k8s_observer,
        expected_metrics=expected_metrics,
        test_timeout=k8s_test_timeout,
    )

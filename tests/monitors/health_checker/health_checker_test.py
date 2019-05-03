import string
from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
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

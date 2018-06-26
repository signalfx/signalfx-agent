from functools import partial as p
from textwrap import dedent
import string

from tests.helpers.util import wait_for, run_agent, run_service, container_ip
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open

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

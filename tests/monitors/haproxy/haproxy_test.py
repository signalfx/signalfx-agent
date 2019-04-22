import string
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.util import container_ip, run_service, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.haproxy, pytest.mark.monitor_with_endpoints]

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

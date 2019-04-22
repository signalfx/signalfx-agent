import string
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_metric_name, tcp_socket_open
from tests.helpers.util import container_ip, run_container, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.consul, pytest.mark.monitor_with_endpoints]

CONSUL_CONFIG = string.Template(
    """
monitors:
  - type: collectd/consul
    host: $host
    port: 8500
    enhancedMetrics: true
"""
)


def test_consul():
    with run_container("consul:0.9.3") as consul_cont:
        host = container_ip(consul_cont)
        config = CONSUL_CONFIG.substitute(host=host)

        assert wait_for(p(tcp_socket_open, host, 8500), 60), "consul service didn't start"

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_metric_name, agent.fake_services, "gauge.consul.catalog.services.total"), 60
            ), "Didn't get consul datapoints"

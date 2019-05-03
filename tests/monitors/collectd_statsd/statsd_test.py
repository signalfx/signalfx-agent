"""
Tests the collectd/statsd monitor
"""
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, has_datapoint_with_metric_name, udp_port_open_locally
from tests.helpers.util import send_udp_message, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.statsd, pytest.mark.monitor_without_endpoints]


def test_statsd_monitor():
    """
    Test basic functionality
    """
    with Agent.run(
        """
monitors:
  - type: collectd/statsd
    listenAddress: localhost
    listenPort: 8125
    counterSum: true
"""
    ) as agent:
        assert wait_for(p(udp_port_open_locally, 8125)), "statsd port never opened!"
        send_udp_message("localhost", 8125, "statsd.[foo=bar,dim=val]test:1|g")

        assert wait_for(
            p(has_datapoint_with_dim, agent.fake_services, "plugin", "statsd")
        ), "Didn't get statsd datapoints"
        assert wait_for(
            p(has_datapoint_with_metric_name, agent.fake_services, "gauge.statsd.test")
        ), "Didn't get statsd.test metric"
        assert wait_for(p(has_datapoint_with_dim, agent.fake_services, "foo", "bar")), "Didn't get foo dimension"

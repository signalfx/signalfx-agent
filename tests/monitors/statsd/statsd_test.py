"""
Tests the statsd monitor
"""
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, has_datapoint_with_metric_name, udp_port_open_locally_netstat
from tests.helpers.util import send_udp_message, wait_for


pytestmark = [pytest.mark.collectd, pytest.mark.statsd, pytest.mark.monitor_without_endpoints]


def test_statsd_monitor():
    """
    Test basic functionality
    """
    with Agent.run(
        """
monitors:
  - type: statsd
    listenAddress: localhost
    listenPort: 8125
"""
    ) as agent:
        assert wait_for(p(udp_port_open_locally_netstat, 8125)), "statsd port never opened!"
        send_udp_message("localhost", 8125, "statsd.test:1|g")

        assert wait_for(
            p(has_datapoint_with_metric_name, agent.fake_services, "statsd.test")
        ), "Didn't get statsd.test metric"


def test_statsd_monitor_prefix():
    """
    Test metric prefix
    """
    with Agent.run(
        """
monitors:
  - type: statsd
    listenAddress: localhost
    listenPort: 8126
    metricPrefix: statsd.appmesh
"""
    ) as agent:
        assert wait_for(p(udp_port_open_locally_netstat, 8126)), "statsd port never opened!"
        send_udp_message(
            "localhost",
            8126,
            "statsd.appmesh.cluster.cds_egress_ecommerce-demo-mesh_gateway-vn_tcp_8080.update_success:8|c",
        )

        assert wait_for(
            p(
                has_datapoint_with_metric_name,
                agent.fake_services,
                "cluster.cds_egress_ecommerce-demo-mesh_gateway-vn_tcp_8080.update_success",
            )
        ), "Didn't get statsd.test metric"


def test_statsd_monitor_conversion():
    """
    Test conversion
    """
    with Agent.run(
        """
monitors:
  - type: statsd
    listenAddress: localhost
    listenPort: 8127
    converters:
    - pattern: 'cluster.cds_{traffic}_{mesh}_{service}-vn_{}.{action}'
      metricName: '{traffic}.{action}'
"""
    ) as agent:
        assert wait_for(p(udp_port_open_locally_netstat, 8127)), "statsd port never opened!"
        send_udp_message(
            "localhost", 8127, "cluster.cds_egress_ecommerce-demo-mesh_gateway-vn_tcp_8080.update_success:8|c"
        )

        assert wait_for(
            p(
                has_datapoint,
                agent.fake_services,
                metric_name="egress.update_success",
                dimensions={
                    "traffic": "egress",
                    "mesh": "ecommerce-demo-mesh",
                    "service": "gateway",
                    "action": "update_success",
                },
            )
        ), "Didn't get metric"

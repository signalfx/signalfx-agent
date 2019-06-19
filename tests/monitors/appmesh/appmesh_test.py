"""
Tests the appmesh monitor
"""
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, udp_port_open_locally
from tests.helpers.util import get_statsd_port, send_udp_message, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.statsd, pytest.mark.monitor_without_endpoints]


def test_appmesh_monitor():
    """
    Test conversion
    """
    with Agent.run(
        """
monitors:
  - type: appmesh
    listenPort: 0
"""
    ) as agent:
        port = get_statsd_port(agent)

        assert wait_for(p(udp_port_open_locally, port)), "statsd port never opened!"
        send_udp_message(
            "localhost",
            port,
            "cluster.cds_egress_ecommerce-demo-mesh_gateway-vn_tcp_8080.upstream_cx_rx_bytes_total:8|c",
        )

        assert wait_for(
            p(
                has_datapoint,
                agent.fake_services,
                metric_name="upstream_cx_rx_bytes_total",
                dimensions={
                    "traffic": "egress",
                    "mesh": "ecommerce-demo-mesh",
                    "service": "gateway",
                    "action": "upstream_cx_rx_bytes_total",
                },
            )
        ), "Didn't get metric"

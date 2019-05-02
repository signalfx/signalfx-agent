"""
Tests the appmesh monitor
"""
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, udp_port_open_locally_netstat
from tests.helpers.util import send_udp_message, get_statsd_port, wait_for


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

        assert wait_for(p(udp_port_open_locally_netstat, port)), "statsd port never opened!"
        send_udp_message(
            "localhost", port, "cluster.cds_egress_ecommerce-demo-mesh_gateway-vn_tcp_8080.update_success:8|c"
        )

        assert wait_for(
            p(
                has_datapoint,
                agent.fake_services,
                metric_name="update_success",
                dimensions={
                    "traffic": "egress",
                    "mesh": "ecommerce-demo-mesh",
                    "service": "gateway",
                    "action": "update_success",
                },
            )
        ), "Didn't get metric"

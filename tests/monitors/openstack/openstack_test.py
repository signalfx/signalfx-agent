import pytest
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip
from tests.helpers.verify import run_agent_verify

pytestmark = [pytest.mark.collectd, pytest.mark.openstack, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/openstack")
DEFAULT_METRICS = METADATA.default_metrics - {
    "gauge.openstack.nova.server.vda_write",
    "gauge.openstack.nova.server.memory-actual",
    "gauge.openstack.nova.server.vda_read_req",
    "gauge.openstack.nova.server.memory-rss",
    "gauge.openstack.nova.server.vda_write_req",
    "gauge.openstack.nova.server.memory",
    "gauge.openstack.nova.server.vda_read",
    # Openstack monitor does not emit any counters
    "counter.openstack.nova.server.rx",
    "counter.openstack.nova.server.rx_packets",
    "counter.openstack.nova.server.tx",
    "counter.openstack.nova.server.tx_packets",
}


@pytest.mark.flaky(reruns=1)
def test_openstacki_default(devstack):
    host = container_ip(devstack)
    run_agent_verify(
        f"""
            monitors:
            - type: collectd/openstack
              authURL: http://{host}/identity/v3
              username: admin
              password: testing123
        """,
        DEFAULT_METRICS,
    )

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
def test_openstack_default(devstack):
    host = container_ip(devstack)
    run_agent_verify(
        f"""
            monitors:
            - type: collectd/openstack
              authURL: http://{host}/identity/v3
              username: admin
              password: testing123
              httpTimeout: 10.001
              requestBatchSize: 10
              queryServerMetrics: true
              queryHypervisorMetrics: true
              novaListServersSearchOpts:
                all_tenants: "TRUE"
                status: "ACTIVE"
        """,
        DEFAULT_METRICS,
    )

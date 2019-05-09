from contextlib import contextmanager
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, http_status, tcp_socket_open, has_datapoint
from tests.helpers.metadata import Metadata
from tests.helpers.util import run_service, wait_for, container_ip
from tests.helpers.verify import verify

pytestmark = [pytest.mark.collectd, pytest.mark.couchbase, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("collectd/couchbase")
LATEST = "enterprise-5.1.0"
VERSIONS = ["enterprise-4.0.0", LATEST]

# TODO: Not sure why these aren't received, investigate at some point.
EXCLUDED = {
    "gauge.nodes.curr_items_tot",
    "gauge.nodes.ops",
    "gauge.nodes.mem_used",
    "gauge.nodes.couch_docs_data_size",
    "gauge.nodes.cmd_get",
    "gauge.nodes.ep_bg_fetched",
    "gauge.nodes.couch_docs_actual_disk_size",
}


# TODO: Test bucket metrics.


@contextmanager
def run_couchbase(tag):
    with run_service(
        "couchbase", buildargs={"COUCHBASE_VERSION": tag}, hostname="node1.cluster"
    ) as couchbase_container:
        host = container_ip(couchbase_container)
        assert wait_for(p(tcp_socket_open, host, 8091), 60), "service not listening on port"
        assert wait_for(
            p(
                http_status,
                url=f"http://{host}:8091/pools/default",
                status=[200],
                username="administrator",
                password="password",
            ),
            120,
        ), "service didn't start"
        yield host


@pytest.mark.flaky(reruns=2)
@pytest.mark.parametrize("tag", VERSIONS)
def test_couchbase_included(tag):
    with run_couchbase(tag) as host, Agent.run(
        f"""
        monitors:
          - type: collectd/couchbase
            host: {host}
            port: 8091
            collectTarget: NODE
            username: administrator
            password: password
        """
    ) as agent:
        verify(agent, (METADATA.metrics_by_group["nodes"] & METADATA.included_metrics) - EXCLUDED)
        assert has_datapoint_with_dim(agent.fake_services, "plugin", "couchbase"), "Didn't get couchbase datapoints"


@pytest.mark.flaky(reruns=2)
def test_couchbase_detailed():
    """Tests that when detailed collect mode is on those metrics are let through"""
    with run_couchbase(LATEST) as host, Agent.run(
        f"""
        monitors:
          - type: collectd/couchbase
            host: {host}
            port: 8091
            collectTarget: NODE
            collectMode: detailed
            username: administrator
            password: password
        """
    ) as agent:

        def test():
            assert wait_for(p(has_datapoint, agent.fake_services, metric_name="gauge.nodes.memoryTotal"))

        wait_for(test)


@pytest.mark.flaky(reruns=2)
def test_couchbase_detailed_extra_metric():
    """Tests that enabling a detailed metric with extra metrics implicitly enables monitor detailed metrics"""
    with run_couchbase(LATEST) as host, Agent.run(
        f"""
        monitors:
          - type: collectd/couchbase
            host: {host}
            port: 8091
            collectTarget: NODE
            username: administrator
            password: password
            extraMetrics: [gauge.nodes.memoryTotal]
        """
    ) as agent:

        def test():
            assert wait_for(p(has_datapoint, agent.fake_services, metric_name="gauge.nodes.memoryTotal"))

        wait_for(test)

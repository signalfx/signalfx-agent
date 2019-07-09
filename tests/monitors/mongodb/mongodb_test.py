from functools import partial as p
from pathlib import Path
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_container, wait_for
from tests.helpers.verify import verify

pytestmark = [pytest.mark.collectd, pytest.mark.mongodb, pytest.mark.monitor_with_endpoints]

SCRIPT_DIR = Path(__file__).parent.resolve()

METADATA = Metadata.from_package("collectd/mongodb")

EXPECTED_DEFAULTS = METADATA.default_metrics - {
    # These metrics only occur on MMAP storage engines.
    "gauge.backgroundFlushing.average_ms",
    "gauge.backgroundFlushing.last_ms",
    "counter.backgroundFlushing.flushes",
    # This one seems to be missing on newer mongo versions
    "gauge.extra_info.heap_usage_bytes",
}


def test_mongo_basic():
    with run_container("mongo:3.6") as mongo_cont:
        host = container_ip(mongo_cont)
        config = dedent(
            f"""
            monitors:
              - type: collectd/mongodb
                host: {host}
                port: 27017
                databases: [admin]
            """
        )
        assert wait_for(p(tcp_socket_open, host, 27017), 60), "service didn't start"

        with Agent.run(config) as agent:
            verify(agent, EXPECTED_DEFAULTS)


def test_mongo_enhanced_metrics():
    with run_container("mongo:3.6") as mongo_cont:
        host = container_ip(mongo_cont)
        config = dedent(
            f"""
            monitors:
              - type: collectd/mongodb
                host: {host}
                port: 27017
                databases: [admin]
                sendCollectionMetrics: true
                sendCollectionTopMetrics: true
            """
        )
        assert wait_for(p(tcp_socket_open, host, 27017), 60), "service didn't start"

        with Agent.run(config) as agent:
            verify(
                agent,
                METADATA.metrics_by_group["collection"]
                | METADATA.metrics_by_group["collection-top"]
                | EXPECTED_DEFAULTS,
            )

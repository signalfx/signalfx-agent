from pathlib import Path
from functools import partial as p

import pytest

from tests.helpers.assertions import tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for
from tests.helpers.verify import run_agent_verify_default_metrics, run_agent_verify_all_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.cassandra, pytest.mark.monitor_with_endpoints]


METADATA = Metadata.from_package("collectd/cassandra")
SCRIPT_DIR = Path(__file__).parent.resolve()


@pytest.mark.flaky(reruns=2)
def test_cassandra_default():
    with run_service("cassandra") as cassandra_cont:
        host = container_ip(cassandra_cont)

        # Wait for the JMX port to be open in the container
        assert wait_for(p(tcp_socket_open, host, 7199)), "Cassandra JMX didn't start"
        run_agent_verify_default_metrics(
            f"""
            monitors:
              - type: collectd/cassandra
                host: {host}
                port: 7199
                username: cassandra
                password: cassandra
            """,
            METADATA,
        )


@pytest.mark.flaky(reruns=2)
def test_cassandra_all():
    with run_service("cassandra") as cassandra_cont:
        host = container_ip(cassandra_cont)

        # Wait for the JMX port to be open in the container
        assert wait_for(p(tcp_socket_open, host, 7199)), "Cassandra JMX didn't start"
        run_agent_verify_all_metrics(
            f"""
            monitors:
              - type: collectd/cassandra
                host: {host}
                port: 7199
                username: cassandra
                password: cassandra
                extraMetrics: ["*"]
            """,
            METADATA,
        )

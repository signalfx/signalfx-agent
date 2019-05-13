from contextlib import contextmanager
from functools import partial as p

from tests.helpers.assertions import tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import run_container, container_ip, wait_for
from tests.helpers.verify import run_agent_verify_included_metrics, run_agent_verify_all_metrics

METADATA = Metadata.from_package("collectd/zookeeper")


@contextmanager
def run_zookeeper():
    with run_container("zookeeper:3.4") as zk_cont:
        host = container_ip(zk_cont)
        assert wait_for(p(tcp_socket_open, host, 2181), 30)
        yield host


def test_zookeeeper():
    with run_zookeeper() as host:
        run_agent_verify_included_metrics(
            f"""
            monitors:
            - type: collectd/zookeeper
              host: {host}
              port: 2181
            """,
            METADATA,
        )


def test_zookeeper_all():
    with run_zookeeper() as host:
        run_agent_verify_all_metrics(
            f"""
            monitors:
            - type: collectd/zookeeper
              host: {host}
              port: 2181
              extraMetrics: ["*"]
            """,
            METADATA,
        )

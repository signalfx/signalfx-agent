from contextlib import contextmanager
from functools import partial as p

from tests.helpers.agent import Agent
from tests.helpers.assertions import tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_container, wait_for
from tests.helpers.verify import run_agent_verify_default_metrics, verify_expected_is_subset

METADATA = Metadata.from_package("collectd/zookeeper")

ENV = ["ZOO_4LW_COMMANDS_WHITELIST=mntr,ruok", "ZOO_STANDALONE_ENABLED=false"]


@contextmanager
def run_zookeeper(version="zookeeper:3.4", env=None):
    with run_container(version, environment=env) as zk_cont:
        host = container_ip(zk_cont)
        assert wait_for(p(tcp_socket_open, host, 2181), 30)
        yield host


def test_zookeeeper():
    with run_zookeeper() as host:
        run_agent_verify_default_metrics(
            f"""
            monitors:
            - type: collectd/zookeeper
              host: {host}
              port: 2181
            """,
            METADATA,
        )


def test_zookeeper_all_common_metrics():
    with run_zookeeper() as host:
        with Agent.run(
            f"""
            monitors:
            - type: collectd/zookeeper
              host: {host}
              port: 2181
              extraMetrics: ["*"]
            """
        ) as agent:
            verify_expected_is_subset(agent, METADATA.all_metrics - METADATA.metrics_by_group["leader"])


def test_zookeeper_leader_metrics():
    with run_zookeeper(version="zookeeper:3.5", env=ENV) as host:
        with Agent.run(
            f"""
            monitors:
            - type: collectd/zookeeper
              host: {host}
              port: 2181
              extraGroups: [leader]
            """
        ) as agent:
            verify_expected_is_subset(agent, METADATA.metrics_by_group["leader"])


def test_zookeeper_latest():
    with run_zookeeper(version="zookeeper:latest", env=ENV) as host:
        run_agent_verify_default_metrics(
            f"""
            monitors:
            - type: collectd/zookeeper
              host: {host}
              port: 2181
            """,
            METADATA,
        )

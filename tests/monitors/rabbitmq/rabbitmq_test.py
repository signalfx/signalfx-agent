from functools import partial as p
from pathlib import Path
from textwrap import dedent

import pika
import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_container, wait_for
from tests.helpers.verify import run_agent_verify

pytestmark = [pytest.mark.collectd, pytest.mark.rabbitmq, pytest.mark.monitor_with_endpoints]

SCRIPT_DIR = Path(__file__).parent.resolve()


def test_rabbitmq():
    with run_container("rabbitmq:3.6-management") as rabbitmq_cont:
        host = container_ip(rabbitmq_cont)
        config = dedent(
            f"""
            monitors:
              - type: collectd/rabbitmq
                host: {host}
                port: 15672
                username: guest
                password: guest
                collectNodes: true
                collectChannels: true
            """
        )

        assert wait_for(p(tcp_socket_open, host, 15672), 60), "service didn't start"

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "rabbitmq")
            ), "Didn't get rabbitmq datapoints"
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin_instance", "%s-15672" % host)
            ), "Didn't get expected plugin_instance dimension"


def test_rabbitmq_broker_name():
    with run_container("rabbitmq:3.6-management") as rabbitmq_cont:
        host = container_ip(rabbitmq_cont)
        assert wait_for(p(tcp_socket_open, host, 15672), 60), "service didn't start"

        with Agent.run(
            dedent(
                f"""
            monitors:
              - type: collectd/rabbitmq
                host: {host}
                brokerName: '{{{{.host}}}}-{{{{.username}}}}'
                port: 15672
                username: guest
                password: guest
                collectNodes: true
                collectChannels: true
            """
            )
        ) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin_instance", "%s-guest" % host)
            ), "Didn't get expected plugin_instance dimension"


METADATA = Metadata.from_package("collectd/rabbitmq")
DEFAULT_METRICS = METADATA.default_metrics - {"counter.queue.message_stats.deliver", "gauge.channel.number"}


def test_rabbitmq_default():
    with run_container("rabbitmq:3.6-management") as rabbitmq_cont:
        host = container_ip(rabbitmq_cont)
        config = dedent(
            f"""
            monitors:
              - type: collectd/rabbitmq
                host: {host}
                port: 15672
                username: guest
                password: guest
            """
        )

        assert wait_for(p(tcp_socket_open, host, 15672), 60), "service didn't start"

        publish_test_message(host)

        run_agent_verify(config, DEFAULT_METRICS)


# Not testing enhanced metrics. Too many metrics are not coming in (over 70)


def publish_test_message(host):
    connection = pika.BlockingConnection(pika.ConnectionParameters(host))
    channel = connection.channel()
    channel.queue_declare(queue="hello")
    channel.basic_publish(exchange="", routing_key="hello", body="hello world")
    channel.close()

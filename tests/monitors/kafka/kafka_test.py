"""
Monitor tests for kafka
"""
import textwrap
from contextlib import contextmanager
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, has_datapoint_with_metric_name, tcp_socket_open
from tests.helpers.util import container_ip, run_container, run_service, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.kafka, pytest.mark.monitor_with_endpoints]


@contextmanager
def run_kafka(version, **kwargs):
    """
    Runs a kafka container with zookeeper
    """
    with run_container("zookeeper:3.5") as zookeeper:
        zkhost = container_ip(zookeeper)
        assert wait_for(p(tcp_socket_open, zkhost, 2181), 60), "zookeeper didn't start"
        with run_service(
            "kafka",
            environment={"KAFKA_ZOOKEEPER_CONNECT": "%s:2181" % (zkhost,), "START_AS": "broker"},
            buildargs={"KAFKA_VERSION": version},
            **kwargs,
        ) as kafka_container:
            kafka_host = container_ip(kafka_container)
            assert wait_for(p(tcp_socket_open, kafka_host, 9092), 60), "kafka broker didn't start"
            assert wait_for(p(tcp_socket_open, kafka_host, 7099), 60), "kafka broker jmx didn't start"
            yield kafka_container


def test_omitting_kafka_metrics(version="1.0.1"):
    with run_kafka(version) as kafka:
        kafka_host = container_ip(kafka)
        with Agent.run(
            textwrap.dedent(
                """
        monitors:
         - type: collectd/kafka
           host: {0}
           port: 7099
           clusterName: testCluster
           mBeansToOmit:
             - kafka-active-controllers
        """.format(
                    kafka_host
                )
            )
        ) as agent:
            assert not wait_for(
                p(has_datapoint_with_metric_name, agent.fake_services, "gauge.kafka-active-controllers"),
                timeout_seconds=60,
            ), "Didn't get kafka datapoints"


VERSIONS = ["0.9.0.0", "0.10.0.0", "0.11.0.0", "1.0.0", "1.0.1", "1.1.1", "2.0.0"]


@pytest.mark.flaky(reruns=2)
@pytest.mark.parametrize("version", VERSIONS)
def test_all_kafka_monitors(version):
    with run_kafka(version) as kafka:
        kafka_host = container_ip(kafka)
        with run_container(
            kafka.image.id,
            environment={"JMX_PORT": "8099", "START_AS": "producer", "KAFKA_BROKER": "%s:9092" % (kafka_host,)},
        ) as kafka_producer:
            kafkaproducerhost = container_ip(kafka_producer)
            assert wait_for(p(tcp_socket_open, kafkaproducerhost, 8099), 60), "kafka producer jmx didn't start"
            with run_container(
                kafka.image.id,
                environment={"JMX_PORT": "9099", "START_AS": "consumer", "KAFKA_BROKER": "%s:9092" % (kafka_host,)},
            ) as kafka_consumer:
                kafkaconsumerhost = container_ip(kafka_consumer)
                assert wait_for(p(tcp_socket_open, kafkaconsumerhost, 9099), 60), "kafka consumer jmx didn't start"
                with Agent.run(
                    textwrap.dedent(
                        """
                monitors:
                 - type: collectd/kafka
                   host: {0}
                   port: 7099
                   clusterName: testCluster
                 - type: collectd/kafka_producer
                   host: {1}
                   port: 8099
                 - type: collectd/kafka_consumer
                   host: {2}
                   port: 9099
                """.format(
                            kafka_host, kafkaproducerhost, kafkaconsumerhost
                        )
                    )
                ) as agent:
                    assert wait_for(
                        p(has_datapoint_with_metric_name, agent.fake_services, "gauge.kafka-active-controllers"),
                        timeout_seconds=60,
                    ), "Didn't get kafka datapoints"
                    assert wait_for(
                        p(has_datapoint_with_dim, agent.fake_services, "cluster", "testCluster"), timeout_seconds=60
                    ), "Didn't get cluster dimension from kafka datapoints"
                    assert wait_for(
                        p(has_datapoint_with_dim, agent.fake_services, "client-id", "console-producer"),
                        timeout_seconds=60,
                    ), "Didn't get client-id dimension from kafka_producer datapoints"
                    assert wait_for(
                        p(has_datapoint_with_dim, agent.fake_services, "client-id", "consumer-1"), timeout_seconds=60
                    ), "Didn't get client-id dimension from kafka_consumer datapoints"

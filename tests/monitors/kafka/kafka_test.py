"""
Monitor tests for kafka
"""
from contextlib import contextmanager
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, has_datapoint_with_metric_name, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import (
    container_ip,
    run_container,
    run_service,
    wait_for,
    DEFAULT_TIMEOUT,
    random_hex,
    container_hostname,
)
from tests.helpers.verify import verify

pytestmark = [pytest.mark.collectd, pytest.mark.kafka, pytest.mark.monitor_with_endpoints]
KAFKA_METADATA = Metadata.from_package("collectd/kafka")
PRODUCER_METADATA = Metadata.from_package("collectd/kafkaproducer")
CONSUMER_METADATA = Metadata.from_package("collectd/kafkaconsumer")
VERSIONS = ["0.9.0.0", "0.10.0.0", "0.11.0.0", "1.0.0", "1.1.1", "2.0.0"]


@contextmanager
def run_kafka(version, **kwargs):
    """
    Runs a kafka container with zookeeper
    """
    args = dict(kwargs)
    args.setdefault("name", f"kafka-broker-{random_hex()}")

    with run_container("zookeeper:3.5") as zookeeper:
        zkhost = container_ip(zookeeper)
        assert wait_for(p(tcp_socket_open, zkhost, 2181), 60), "zookeeper didn't start"
        with run_service(
            "kafka",
            environment={f"KAFKA_ZOOKEEPER_CONNECT": f"{zkhost}:2181", "START_AS": "broker"},
            buildargs={"KAFKA_VERSION": version},
            **args,
        ) as kafka_container:
            kafka_host = container_ip(kafka_container)
            assert wait_for(p(tcp_socket_open, kafka_host, 9092), 60), "kafka broker didn't start"
            assert wait_for(p(tcp_socket_open, kafka_host, 7099), 60), "kafka broker jmx didn't start"

            with run_container(
                kafka_container.image.id,
                environment={"START_AS": "create-topic", f"KAFKA_ZOOKEEPER_CONNECT": f"{zkhost}:2181"},
            ) as kafka_topic:
                assert kafka_topic.wait(timeout=DEFAULT_TIMEOUT)["StatusCode"] == 0, "unable to create kafka topic"

            yield kafka_container


@contextmanager
def run_producer(image, kafka_host, **kwargs):
    with run_container(
        image,
        name=f"kafka-producer-{random_hex()}",
        environment={"JMX_PORT": "8099", "START_AS": "producer", "KAFKA_BROKER": f"{kafka_host}:9092"},
        **kwargs,
    ) as kafka_producer:
        host = container_ip(kafka_producer)
        assert wait_for(p(tcp_socket_open, host, 8099), 60), "kafka producer jmx didn't start"
        yield host


@contextmanager
def run_consumer(image, kafka_host, **kwargs):
    with run_container(
        image,
        name=f"kafka-consumer-{random_hex()}",
        environment={"JMX_PORT": "9099", "START_AS": "consumer", "KAFKA_BROKER": f"{kafka_host}:9092"},
        **kwargs,
    ) as kafka_consumer:
        host = container_ip(kafka_consumer)
        assert wait_for(p(tcp_socket_open, host, 9099), 60), "kafka consumer jmx didn't start"
        yield host


def test_omitting_kafka_metrics(version="1.0.1"):
    with run_kafka(version) as kafka:
        kafka_host = container_ip(kafka)
        with Agent.run(
            f"""
            monitors:
             - type: collectd/kafka
               host: {kafka_host}
               port: 7099
               clusterName: testCluster
               mBeansToOmit:
                 - kafka-active-controllers
            """
        ) as agent:
            assert not wait_for(
                p(has_datapoint_with_metric_name, agent.fake_services, "gauge.kafka-active-controllers"),
                timeout_seconds=60,
            ), "Didn't get kafka datapoints"


def run_all(version, metrics, extra_metrics=""):
    with run_kafka(version) as kafka:
        kafka_ip = container_ip(kafka)
        kafka_host = container_hostname(kafka)

        image = kafka.image.id

        # We add the Kafka broker host:ip as an extra_host because by default the Kafka broker advertises itself with
        # its hostname and without this the producer and consumer wouldn't be able to resolve the broker hostname.
        with run_producer(image, kafka_host, extra_hosts={kafka_host: kafka_ip}) as kafkaproducerhost, run_consumer(
            image, kafka_host, extra_hosts={kafka_host: kafka_ip}
        ) as kafkaconsumerhost, Agent.run(
            f"""
            monitors:
             - type: collectd/kafka
               host: {kafka_ip}
               port: 7099
               clusterName: testCluster
               extraMetrics: {extra_metrics}
             - type: collectd/kafka_producer
               host: {kafkaproducerhost}
               port: 8099
               extraMetrics: {extra_metrics}
             - type: collectd/kafka_consumer
               host: {kafkaconsumerhost}
               port: 9099
               extraMetrics: {extra_metrics}
            """
        ) as agent:
            verify(agent, metrics)
            assert has_datapoint_with_dim(
                agent.fake_services, "cluster", "testCluster"
            ), "Didn't get cluster dimension from kafka datapoints"
            assert has_datapoint_with_dim(
                agent.fake_services, "client-id", "console-producer"
            ), "Didn't get client-id dimension from kafka_producer datapoints"
            assert has_datapoint_with_dim(
                agent.fake_services, "client-id", "consumer-1"
            ), "Didn't get client-id dimension from kafka_consumer datapoints"


@pytest.mark.flaky(reruns=2)
@pytest.mark.parametrize("version", VERSIONS)
def test_kafka_monitors_included(version):
    run_all(
        version,
        KAFKA_METADATA.included_metrics | PRODUCER_METADATA.included_metrics | CONSUMER_METADATA.included_metrics,
    )


@pytest.mark.flaky(reruns=2)
@pytest.mark.parametrize("version", VERSIONS)
def test_kafka_monitors_all(version):
    run_all(
        version,
        KAFKA_METADATA.all_metrics | PRODUCER_METADATA.all_metrics | CONSUMER_METADATA.all_metrics,
        extra_metrics="['*']",
    )

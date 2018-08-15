"""
Monitor tests for kafka
"""
import os
import textwrap
from contextlib import contextmanager
from functools import partial as p

import pytest

from tests.helpers.assertions import (has_datapoint_with_metric_name, has_datapoint_with_dim, tcp_socket_open)
from tests.helpers.util import (
    container_ip, get_monitor_dims_from_selfdescribe, get_monitor_metrics_from_selfdescribe, run_agent, run_container,
    run_service, wait_for
)
from tests.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test

pytestmark = [pytest.mark.collectd, pytest.mark.kafka, pytest.mark.monitor_with_endpoints]


@contextmanager
def run_kafka(version):
    """
    Runs a kafka container with zookeeper
    """
    with run_container("zookeeper:3.5") as zookeeper:
        zkhost = container_ip(zookeeper)
        assert wait_for(p(tcp_socket_open, zkhost, 2181), 60), "zookeeper didn't start"
        with run_service(
            "kafka",
            environment={"JMX_PORT": "7099", "KAFKA_ZOOKEEPER_CONNECT": "%s:2181" % (zkhost,),
                         "START_AS": "broker"},
            buildargs={"KAFKA_VERSION": version}
        ) as kafka_container:
            run_service(
                "kafka",
                environment={"START_AS": "create-topic", "KAFKA_ZOOKEEPER_CONNECT": "%s:2181" % (zkhost,)},
                buildargs={"KAFKA_VERSION": version}
            )
            yield kafka_container


def test_omitting_kafka_metrics(version="1.0.1"):
    with run_kafka(version) as kafka:
        kafkahost = container_ip(kafka)
        with run_agent(textwrap.dedent("""
        monitors:
         - type: collectd/kafka
           host: {0}
           port: 7099
           clusterName: testCluster
           mBeansToOmit:
             - kafka-active-controllers
        """.format(kafkahost))) as [backend, _, _]:
            assert not wait_for(p(has_datapoint_with_metric_name, backend, "gauge.kafka-active-controllers"),
                              timeout_seconds=60), "Didn't get kafka datapoints"


versions = ["0.9.0.0", "0.10.0", "0.11.0", "1.0.0", "1.0.1", "1.1.1"]

@pytest.mark.flaky(reruns=2)
@pytest.mark.parametrize("version", versions)
def test_all_kafka_monitors(version):
    with run_kafka(version) as kafka:
        kafkahost = container_ip(kafka)
        with run_service(
            "kafka",
            environment={"JMX_PORT": "8099", "START_AS": "producer", "KAFKA_BROKER": "%s:9092" % (kafkahost,)},
            buildargs={"KAFKA_VERSION": version}
        ) as kafka_producer:
            kafkaproducerhost = container_ip(kafka_producer)
            with run_service(
                "kafka",
                environment={"JMX_PORT": "9099", "START_AS": "consumer", "KAFKA_BROKER": "%s:9092" % (kafkahost,)},
                buildargs={"KAFKA_VERSION": version}
                ) as kafka_consumer:
                kafkaconsumerhost = container_ip(kafka_consumer)
                with run_agent(textwrap.dedent("""
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
                """.format(kafkahost, kafkaproducerhost, kafkaconsumerhost))) as [backend, _, _]:
                    assert wait_for(p(has_datapoint_with_metric_name, backend, "gauge.kafka-active-controllers"),
                                      timeout_seconds=60), "Didn't get kafka datapoints"
                    assert wait_for(p(has_datapoint_with_dim, backend, "cluster", "testCluster"),
                                      timeout_seconds=60), "Didn't get cluster dimension from kafka datapoints"
                    assert wait_for(p(has_datapoint_with_dim, backend, "client-id", "console-producer"),
                                      timeout_seconds=60), "Didn't get client-id dimension from kafka_producer datapoints"
                    assert wait_for(p(has_datapoint_with_dim, backend, "client-id", "consumer-1"),
                                      timeout_seconds=60), "Didn't get client-id dimension from kafka_consumer datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_kafka_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = os.path.join(os.path.dirname(os.path.realpath(__file__)), "kafka-k8s.yaml")
    monitors = [
        {
            "type": "collectd/kafka",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
            "serviceURL": 'service:jmx:rmi:///jndi/rmi://{{.Host}}:{{.Port}}/jmxrmi',
            "username": "testuser",
            "password": "testing123",
            "clusterName": "testcluster"
        },
    ]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=[yaml],
        observer=k8s_observer,
        expected_metrics=get_monitor_metrics_from_selfdescribe(monitors[0]["type"]),
        expected_dims=get_monitor_dims_from_selfdescribe(monitors[0]["type"]),
        test_timeout=k8s_test_timeout
    )

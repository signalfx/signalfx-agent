"""
Monitor tests for kafka
"""
import os
import textwrap
from contextlib import contextmanager
from functools import partial as p

import pytest

from tests.helpers.assertions import (has_datapoint_with_metric_name, tcp_socket_open)
from tests.helpers.util import (
    container_ip, get_monitor_dims_from_selfdescribe, get_monitor_metrics_from_selfdescribe, run_agent, run_container,
    run_service, wait_for
)
from tests.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test

pytestmark = [pytest.mark.collectd, pytest.mark.kafka, pytest.mark.monitor_with_endpoints]


@contextmanager
def run_kafka(version="1.0.1"):
    """
    Runs a kafka container with zookeeper
    """
    with run_container("zookeeper:3.5") as zookeeper:
        zkhost = container_ip(zookeeper)
        assert wait_for(p(tcp_socket_open, zkhost, 2181), 60), "zookeeper didn't start"
        with run_service(
            "kafka",
            environment={"KAFKA_ZOOKEEPER_CONNECT": "%s:2181" % (zkhost,)},
            buildargs={"KAFKA_VERSION": version}
        ) as kafka_container:
            yield kafka_container


def test_kafka_monitor():
    with run_kafka() as kafka:
        with run_agent(textwrap.dedent("""
        monitors:
         - type: collectd/kafka
           host: {0}
           port: 1099
        """.format(container_ip(kafka)))) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_metric_name, backend, "gauge.kafka-active-controllers")), \
                    "Didn't get kafka datapoints"


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
            "password": "testing123"
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

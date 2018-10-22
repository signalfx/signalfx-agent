from functools import partial as p
from textwrap import dedent
import os
import pytest
import semver

from helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test
from helpers.util import (
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_container,
    run_agent,
    container_ip,
    wait_for,
)

pytestmark = [pytest.mark.collectd, pytest.mark.elasticsearch, pytest.mark.monitor_with_endpoints]


def test_elasticsearch():
    es_env = {"ELASTIC_PASSWORD": "testing123", "discovery.type": "single-node", "ES_JAVA_OPTS": "-Xms128m -Xmx128m"}
    with run_container("docker.elastic.co/elasticsearch/elasticsearch:6.2.4", environment=es_env) as es_container:
        host = container_ip(es_container)
        assert wait_for(p(tcp_socket_open, host, 9200), 90), "service didn't start"
        config = dedent(
            f"""
            monitors:
            - type: collectd/elasticsearch
              host: {host}
              port: 9200
              username: elastic
              password: testing123
            """
        )
        with run_agent(config) as [backend, _, _]:
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "elasticsearch")
            ), "Didn't get elasticsearch datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_elasticsearch_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    if semver.match(minikube.k8s_version.lstrip("v"), "<1.8.0"):
        pytest.skip('required env var "discovery.type" for elasticsearch not supported in K8S versions < 1.8.0')
    yaml = os.path.join(os.path.dirname(os.path.realpath(__file__)), "elasticsearch-k8s.yaml")
    monitors = [
        {
            "type": "collectd/elasticsearch",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
            "detailedMetrics": True,
            "username": "elastic",
            "password": "testing123",
        }
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
        test_timeout=k8s_test_timeout,
    )

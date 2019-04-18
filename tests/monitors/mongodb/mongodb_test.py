from functools import partial as p
from pathlib import Path
from textwrap import dedent

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test
from tests.helpers.util import (
    container_ip,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_container,
    wait_for,
)

pytestmark = [pytest.mark.collectd, pytest.mark.mongodb, pytest.mark.monitor_with_endpoints]

DIR = Path(__file__).parent.resolve()


def test_mongo():
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
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"plugin": "mongo"})
            ), "Didn't get mongo datapoints"


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
            assert wait_for(
                p(has_datapoint, agent.fake_services, metric_name="gauge.collection.size"), 60
            ), "Did not get datapoint from SendCollectionMetrics config"
            assert wait_for(
                p(has_datapoint, agent.fake_services, metric_name="counter.collection.commandsTime"), 60
            ), "Did not get datapoint from SendCollectionTopMetrics config"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_mongodb_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = DIR / "mongodb-k8s.yaml"
    monitors = [
        {
            "type": "collectd/mongodb",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
            "databases": ["admin"],
            "username": "testuser",
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

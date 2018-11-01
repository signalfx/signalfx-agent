from functools import partial as p
from textwrap import dedent
import os
import pytest

from helpers.assertions import has_datapoint_with_dim, http_status, has_log_message
from helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test
from helpers.util import (
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_service,
    run_agent,
    container_ip,
    wait_for,
)

pytestmark = [pytest.mark.collectd, pytest.mark.elasticsearch, pytest.mark.monitor_with_endpoints]


@pytest.mark.flaky(reruns=2)
def test_elasticsearch_without_cluster_option():
    with run_service("elasticsearch", environment={"cluster.name": "testCluster"}) as es_container:
        host = container_ip(es_container)
        assert wait_for(
            p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
        ), "service didn't start"
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
        with run_agent(config) as [backend, get_output, _]:
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "elasticsearch")
            ), "Didn't get elasticsearch datapoints"
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin_instance", "testCluster")
            ), "Cluster name not picked from read callback"
            assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"


@pytest.mark.flaky(reruns=2)
def test_elasticsearch_with_cluster_option():
    with run_service("elasticsearch", environment={"cluster.name": "testCluster"}) as es_container:
        host = container_ip(es_container)
        assert wait_for(
            p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
        ), "service didn't start"
        config = dedent(
            f"""
            monitors:
            - type: collectd/elasticsearch
              host: {host}
              port: 9200
              username: elastic
              password: testing123
              cluster: testCluster1
            """
        )
        with run_agent(config) as [backend, get_output, _]:
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "elasticsearch")
            ), "Didn't get elasticsearch datapoints"
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin_instance", "testCluster1")
            ), "Cluster name not picked from read callback"
            # make sure all plugin_instance dimensions were overridden by the cluster option
            assert not wait_for(
                p(has_datapoint_with_dim, backend, "plugin_instance", "testCluster"), 10
            ), "plugin_instance dimension not overridden by cluster option"
            assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"


# To mimic the scenario where node is not up
@pytest.mark.flaky(reruns=2)
def test_elasticsearch_without_cluster():
    # start the ES container without the service
    with run_service(
        "elasticsearch", environment={"cluster.name": "testCluster"}, entrypoint="sleep inf"
    ) as es_container:
        host = container_ip(es_container)
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
            assert not wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "elasticsearch")
            ), "datapoints found without service"
            # start ES service and make sure it gets discovered
            es_container.exec_run("/usr/local/bin/docker-entrypoint.sh eswrapper", detach=True)
            assert wait_for(
                p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
            ), "service didn't start"
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "elasticsearch")
            ), "Didn't get elasticsearch datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_elasticsearch_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = os.path.join(os.path.dirname(os.path.realpath(__file__)), "elasticsearch-k8s.yaml")
    dockerfile_dir = os.path.join(os.path.dirname(os.path.realpath(__file__)), "../../../test-services/elasticsearch")
    build_opts = {"tag": "elasticsearch:k8s-test"}
    minikube.build_image(dockerfile_dir, build_opts)
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

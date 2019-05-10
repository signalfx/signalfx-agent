from functools import partial as p
from textwrap import dedent

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import any_metric_has_any_dim_key, has_datapoint_with_dim, has_log_message, http_status
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for
from tests.helpers.verify import run_agent_verify

pytestmark = [pytest.mark.collectd, pytest.mark.elasticsearch, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("elasticsearch")
ENV = {"cluster.name": "testCluster"}
AGENT_CONFIG_TEMPLATE = dedent(
    """
    monitors:
    - type: elasticsearch
      host: {host}
      port: 9200
      username: elastic
      password: testing123
      {flag}
    """
)


def check_service_status(host):
    assert wait_for(p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180), "service didn't start"


@pytest.mark.flaky(reruns=2)
def test_elasticsearch_without_cluster_option():
    with run_service("elasticsearch/6.4.2", environment=ENV) as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, flag="")
        with Agent.run(agent_config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "elasticsearch")
            ), "Didn't get elasticsearch datapoints"
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin_instance", "testCluster")
            ), "Cluster name not picked from read callback"
            assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


@pytest.mark.flaky(reruns=2)
def test_elasticsearch_with_cluster_option():
    with run_service("elasticsearch/6.4.2", environment=ENV) as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, flag="cluster: testCluster1")
        with Agent.run(agent_config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "elasticsearch")
            ), "Didn't get elasticsearch datapoints"
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin_instance", "testCluster1")
            ), "Cluster name not picked from read callback"
            # make sure all plugin_instance dimensions were overridden by the cluster option
            assert not wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin_instance", "testCluster"), 10
            ), "plugin_instance dimension not overridden by cluster option"
            assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


# To mimic the scenario where node is not up
@pytest.mark.flaky(reruns=2)
def test_elasticsearch_without_cluster():
    # start the ES container without the service
    with run_service("elasticsearch/6.4.2", environment=ENV, entrypoint="sleep inf") as es_container:
        host = container_ip(es_container)
        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, flag="")
        with Agent.run(agent_config) as agent:
            assert not wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "elasticsearch")
            ), "datapoints found without service"
            # start ES service and make sure it gets discovered
            es_container.exec_run("/usr/local/bin/docker-entrypoint.sh eswrapper", detach=True)
            assert wait_for(
                p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
            ), "service didn't start"


@pytest.mark.flaky(reruns=2)
def test_with_default_config_6_6_1():
    with run_service("elasticsearch/6.6.1") as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, flag="")
        with Agent.run(agent_config) as agent:
            assert wait_for(
                p(any_metric_has_any_dim_key, agent.fake_services, METADATA.included_metrics, METADATA.dims)
            ), "Didn't get all default dimensions"
            assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


@pytest.mark.flaky(reruns=2)
def test_with_default_config_2_4_5():
    with run_service("elasticsearch/2.4.5") as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, flag="")
        with Agent.run(agent_config) as agent:
            assert wait_for(
                p(any_metric_has_any_dim_key, agent.fake_services, METADATA.included_metrics, METADATA.dims)
            ), "Didn't get all default dimensions"
            assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


@pytest.mark.flaky(reruns=2)
def test_with_default_config_2_0_2():
    with run_service("elasticsearch/2.0.2") as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, flag="")
        with Agent.run(agent_config) as agent:
            assert wait_for(
                p(any_metric_has_any_dim_key, agent.fake_services, METADATA.included_metrics, METADATA.dims)
            ), "Didn't get all default dimensions"
            assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


@pytest.mark.flaky(reruns=2)
def test_elasticsearch_with_enhanced_cluster_health_stats():
    expected_metrics = METADATA.included_metrics | METADATA.metrics_by_group["cluster"]
    with run_service("elasticsearch/6.4.2", environment=ENV) as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, flag="enableEnhancedClusterHealthStats: true")
        run_agent_verify(agent_config, expected_metrics)


@pytest.mark.flaky(reruns=2)
def test_elasticsearch_with_enhanced_http_stats():
    expected_metrics = METADATA.included_metrics | METADATA.metrics_by_group["node/http"]
    with run_service("elasticsearch/6.4.2", environment=ENV) as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, flag="enableEnhancedHTTPStats: true")
        run_agent_verify(agent_config, expected_metrics)


@pytest.mark.flaky(reruns=2)
def test_elasticsearch_with_enhanced_jvm_stats():
    expected_metrics = METADATA.included_metrics | METADATA.metrics_by_group["node/jvm"]
    with run_service("elasticsearch/6.4.2", environment=ENV) as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, flag="enableEnhancedJVMStats: true")
        run_agent_verify(agent_config, expected_metrics)


@pytest.mark.flaky(reruns=2)
def test_elasticsearch_with_enhanced_process_stats():
    expected_metrics = METADATA.included_metrics | METADATA.metrics_by_group["node/process"]
    with run_service("elasticsearch/6.4.2", environment=ENV) as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, flag="enableEnhancedProcessStats: true")
        run_agent_verify(agent_config, expected_metrics)


@pytest.mark.flaky(reruns=2)
def test_elasticsearch_with_enhanced_thread_pool_stats():
    expected_metrics = METADATA.included_metrics | METADATA.metrics_by_group["node/thread-pool"]
    with run_service("elasticsearch/6.4.2", environment=ENV) as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, flag="enableEnhancedThreadPoolStats: true")
        run_agent_verify(agent_config, expected_metrics)


@pytest.mark.flaky(reruns=2)
def test_elasticsearch_with_enhanced_transport_stats():
    expected_metrics = METADATA.included_metrics | METADATA.metrics_by_group["node/transport"]
    with run_service("elasticsearch/6.4.2", environment=ENV) as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, flag="enableEnhancedTransportStats: true")
        run_agent_verify(agent_config, expected_metrics)


@pytest.mark.flaky(reruns=2)
def test_elasticsearch_all_metrics():
    with run_service("elasticsearch/6.4.2", environment=ENV) as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        es_6_4_2_expected_metrics = METADATA.all_metrics - {
            "elasticsearch.indices.percolate.queries",
            "elasticsearch.indices.percolate.total",
            "elasticsearch.indices.percolate.time",
            "elasticsearch.indices.filter-cache.memory-size",
            "elasticsearch.indices.id-cache.memory-size",
            "elasticsearch.indices.percolate.current",
            "elasticsearch.indices.suggest.current",
            "elasticsearch.indices.suggest.time",
            "elasticsearch.indices.store.throttle-time",
            "elasticsearch.indices.suggest.total",
            "elasticsearch.indices.filter-cache.evictions",
            "elasticsearch.indices.segments.index-writer-max-memory-size",
        }
        config = dedent(
            f"""
            monitors:
            - type: elasticsearch
              host: {host}
              port: 9200
              username: elastic
              password: testing123
              enableEnhancedClusterHealthStats: true
              enableEnhancedHTTPStats: true
              enableEnhancedJVMStats: true
              enableEnhancedProcessStats: true
              enableEnhancedThreadPoolStats: true
              enableEnhancedTransportStats: true
              enableEnhancedNodeIndicesStats:
              - docs
              - store
              - indexing
              - get
              - search
              - merges
              - refresh
              - flush
              - warmer
              - query_cache
              - filter_cache
              - fielddata
              - completion
              - segments
              - translog
              - request_cache
              - recovery
              - id_cache
              - suggest
              - percolate
              enableEnhancedIndexStatsForIndexGroups:
              - docs
              - store
              - indexing
              - get
              - search
              - merges
              - refresh
              - flush
              - warmer
              - query_cache
              - filter_cache
              - fielddata
              - completion
              - segments
              - translog
              - request_cache
              - recovery
              - id_cache
              - suggest
              - percolate
            """
        )
        run_agent_verify(config, es_6_4_2_expected_metrics)

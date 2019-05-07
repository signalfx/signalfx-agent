from functools import partial as p
from textwrap import dedent

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import any_metric_has_any_dim_key, has_datapoint_with_dim, has_log_message, http_status
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for
from tests.helpers.verify import verify_custom

pytestmark = [pytest.mark.collectd, pytest.mark.elasticsearch, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("elasticsearch")


@pytest.mark.flaky(reruns=0)
def test_elasticsearch_without_cluster_option():
    with run_service("elasticsearch/6.4.2", environment={"cluster.name": "testCluster"}) as es_container:
        host = container_ip(es_container)
        assert wait_for(
            p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
        ), "service didn't start"
        config = dedent(
            f"""
            monitors:
            - type: elasticsearch
              host: {host}
              port: 9200
              username: elastic
              password: testing123
            """
        )
        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "elasticsearch")
            ), "Didn't get elasticsearch datapoints"
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin_instance", "testCluster")
            ), "Cluster name not picked from read callback"
            assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


@pytest.mark.flaky(reruns=0)
def test_elasticsearch_with_cluster_option():
    with run_service("elasticsearch/6.4.2", environment={"cluster.name": "testCluster"}) as es_container:
        host = container_ip(es_container)
        assert wait_for(
            p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
        ), "service didn't start"
        config = dedent(
            f"""
            monitors:
            - type: elasticsearch
              host: {host}
              port: 9200
              username: elastic
              password: testing123
              cluster: testCluster1
            """
        )
        with Agent.run(config) as agent:
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
@pytest.mark.flaky(reruns=0)
def test_elasticsearch_without_cluster():
    # start the ES container without the service
    with run_service(
        "elasticsearch/6.4.2", environment={"cluster.name": "testCluster"}, entrypoint="sleep inf"
    ) as es_container:
        host = container_ip(es_container)
        config = dedent(
            f"""
            monitors:
            - type: elasticsearch
              host: {host}
              port: 9200
              username: elastic
              password: testing123
            """
        )
        with Agent.run(config) as agent:
            assert not wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "elasticsearch")
            ), "datapoints found without service"
            # start ES service and make sure it gets discovered
            es_container.exec_run("/usr/local/bin/docker-entrypoint.sh eswrapper", detach=True)
            assert wait_for(
                p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
            ), "service didn't start"


# TODO: fix this failing test
# @pytest.mark.flaky(reruns=0)
# def test_elasticsearch_with_threadpool():
#     with run_service("elasticsearch/6.4.2", environment={"cluster.name": "testCluster"}) as es_container:
#         host = container_ip(es_container)
#         assert wait_for(
#             p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
#         ), "service didn't start"
#         config = dedent(
#             f"""
#             monitors:
#             - type: elasticsearch
#               host: {host}
#               port: 9200
#               username: elastic
#               password: testing123
#               threadPools:
#               - bulk
#               - index
#               - search
#             """
#         )
#     with Agent.run(config) as agent:
#             assert wait_for(
#                 p(has_datapoint_with_dim, agent.fake_services, "plugin", "elasticsearch")
#             ), "Didn't get elasticsearch datapoints"
#             assert wait_for(
#                 p(has_datapoint_with_dim, agent.fake_services, "thread_pool", "bulk")
#             ), "Didn't get bulk thread pool metrics"
#             assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


DEFAULT_METRICS = METADATA.included_metrics

DEFAULT_DIMENSIONS = METADATA.dims


@pytest.mark.flaky(reruns=0)
def test_with_default_config_6_6_1():
    with run_service("elasticsearch/6.6.1") as es_container:
        host = container_ip(es_container)
        assert wait_for(
            p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
        ), "service didn't start"
        config = dedent(
            f"""
            monitors:
            - type: elasticsearch
              host: {host}
              port: 9200
            """
        )
        with Agent.run(config) as agent:
            assert wait_for(
                p(any_metric_has_any_dim_key, agent.fake_services, DEFAULT_METRICS, DEFAULT_DIMENSIONS)
            ), "Didn't get all default dimensions"
            assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


@pytest.mark.flaky(reruns=0)
def test_with_default_config_2_4_5():
    with run_service("elasticsearch/2.4.5") as es_container:
        host = container_ip(es_container)
        assert wait_for(
            p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
        ), "service didn't start"
        config = dedent(
            f"""
            monitors:
            - type: elasticsearch
              host: {host}
              port: 9200
            """
        )
        with Agent.run(config) as agent:
            assert wait_for(
                p(any_metric_has_any_dim_key, agent.fake_services, DEFAULT_METRICS, DEFAULT_DIMENSIONS)
            ), "Didn't get all default dimensions"
            assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


@pytest.mark.flaky(reruns=0)
def test_with_default_config_2_0_2():
    with run_service("elasticsearch/2.0.2") as es_container:
        host = container_ip(es_container)
        assert wait_for(
            p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
        ), "service didn't start"
        config = dedent(
            f"""
            monitors:
            - type: elasticsearch
              host: {host}
              port: 9200
            """
        )
        with Agent.run(config) as agent:
            assert wait_for(
                p(any_metric_has_any_dim_key, agent.fake_services, DEFAULT_METRICS, DEFAULT_DIMENSIONS)
            ), "Didn't get all default dimensions"
            assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


@pytest.mark.flaky(reruns=0)
def test_elasticsearch_with_enhanced_cluster_health_stats():
    with run_service("elasticsearch/6.4.2", environment={"cluster.name": "testCluster"}) as es_container:
        host = container_ip(es_container)
        assert wait_for(
            p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
        ), "service didn't start"
        config = dedent(
            f"""
            monitors:
            - type: elasticsearch
              host: {host}
              port: 9200
              username: elastic
              password: testing123
              enableEnhancedClusterHealthStats: true
            """
        )
        expected_metrics = DEFAULT_METRICS | {
            "elasticsearch.cluster.active-primary-shards",
            "elasticsearch.cluster.active-shards",
            "elasticsearch.cluster.active-shards-percent",
            "elasticsearch.cluster.delayed-unassigned-shards",
            "elasticsearch.cluster.in-flight-fetches",
            "elasticsearch.cluster.initializing-shards",
            "elasticsearch.cluster.number-of-data_nodes",
            "elasticsearch.cluster.number-of-nodes",
            "elasticsearch.cluster.pending-tasks",
            "elasticsearch.cluster.relocating-shards",
            "elasticsearch.cluster.status",
            "elasticsearch.cluster.task-max-wait-time",
            "elasticsearch.cluster.unassigned-shards",
        }
        verify_custom(config, expected_metrics)


@pytest.mark.flaky(reruns=0)
def test_elasticsearch_with_enhanced_http_stats():
    with run_service("elasticsearch/6.4.2", environment={"cluster.name": "testCluster"}) as es_container:
        host = container_ip(es_container)
        assert wait_for(
            p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
        ), "service didn't start"
        config = dedent(
            f"""
            monitors:
            - type: elasticsearch
              host: {host}
              port: 9200
              username: elastic
              password: testing123
              enableEnhancedHTTPStats: true
            """
        )
        expected_metrics = DEFAULT_METRICS | {"elasticsearch.http.current_open", "elasticsearch.http.total_open"}
        verify_custom(config, expected_metrics)


@pytest.mark.flaky(reruns=0)
def test_elasticsearch_with_enhanced_jvm_stats():
    with run_service("elasticsearch/6.4.2", environment={"cluster.name": "testCluster"}) as es_container:
        host = container_ip(es_container)
        assert wait_for(
            p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
        ), "service didn't start"
        config = dedent(
            f"""
            monitors:
            - type: elasticsearch
              host: {host}
              port: 9200
              username: elastic
              password: testing123
              enableEnhancedJVMStats: true
            """
        )
        expected_metrics = DEFAULT_METRICS | {
            "elasticsearch.jvm.classes.current-loaded-count",
            "elasticsearch.jvm.classes.total-loaded-count",
            "elasticsearch.jvm.classes.total-unloaded-count",
            "elasticsearch.jvm.gc.count",
            "elasticsearch.jvm.gc.old-count",
            "elasticsearch.jvm.gc.old-time",
            "elasticsearch.jvm.gc.time",
            "elasticsearch.jvm.mem.buffer_pools.direct.count",
            "elasticsearch.jvm.mem.buffer_pools.direct.total_capacity_in_bytes",
            "elasticsearch.jvm.mem.buffer_pools.direct.used_in_bytes",
            "elasticsearch.jvm.mem.buffer_pools.mapped.count",
            "elasticsearch.jvm.mem.buffer_pools.mapped.total_capacity_in_bytes",
            "elasticsearch.jvm.mem.buffer_pools.mapped.used_in_bytes",
            "elasticsearch.jvm.mem.heap-committed",
            "elasticsearch.jvm.mem.heap-max",
            "elasticsearch.jvm.mem.heap-used",
            "elasticsearch.jvm.mem.heap-used-percent",
            "elasticsearch.jvm.mem.non-heap-committed",
            "elasticsearch.jvm.mem.non-heap-used",
            "elasticsearch.jvm.mem.pools.old.max_in_bytes",
            "elasticsearch.jvm.mem.pools.old.peak_max_in_bytes",
            "elasticsearch.jvm.mem.pools.old.peak_used_in_bytes",
            "elasticsearch.jvm.mem.pools.old.used_in_bytes",
            "elasticsearch.jvm.mem.pools.survivor.max_in_bytes",
            "elasticsearch.jvm.mem.pools.survivor.peak_max_in_bytes",
            "elasticsearch.jvm.mem.pools.survivor.peak_used_in_bytes",
            "elasticsearch.jvm.mem.pools.survivor.used_in_bytes",
            "elasticsearch.jvm.mem.pools.young.max_in_bytes",
            "elasticsearch.jvm.mem.pools.young.peak_max_in_bytes",
            "elasticsearch.jvm.mem.pools.young.peak_used_in_bytes",
            "elasticsearch.jvm.mem.pools.young.used_in_bytes",
            "elasticsearch.jvm.threads.count",
            "elasticsearch.jvm.threads.peak",
            "elasticsearch.jvm.uptime",
        }
        verify_custom(config, expected_metrics)


@pytest.mark.flaky(reruns=0)
def test_elasticsearch_with_enhanced_process_stats():
    with run_service("elasticsearch/6.4.2", environment={"cluster.name": "testCluster"}) as es_container:
        host = container_ip(es_container)
        assert wait_for(
            p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
        ), "service didn't start"
        config = dedent(
            f"""
            monitors:
            - type: elasticsearch
              host: {host}
              port: 9200
              username: elastic
              password: testing123
              enableEnhancedProcessStats: true
            """
        )
        expected_metrics = DEFAULT_METRICS | {
            "elasticsearch.process.cpu.percent",
            "elasticsearch.process.cpu.time",
            "elasticsearch.process.max_file_descriptors",
            "elasticsearch.process.mem.total-virtual-size",
            "elasticsearch.process.open_file_descriptors",
        }
        verify_custom(config, expected_metrics)


@pytest.mark.flaky(reruns=0)
def test_elasticsearch_with_enhanced_thread_pool_stats():
    with run_service("elasticsearch/6.4.2", environment={"cluster.name": "testCluster"}) as es_container:
        host = container_ip(es_container)
        assert wait_for(
            p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
        ), "service didn't start"
        config = dedent(
            f"""
            monitors:
            - type: elasticsearch
              host: {host}
              port: 9200
              username: elastic
              password: testing123
              enableEnhancedThreadPoolStats: true
            """
        )
        expected_metrics = DEFAULT_METRICS | {
            "elasticsearch.thread_pool.active",
            "elasticsearch.thread_pool.completed",
            "elasticsearch.thread_pool.largest",
            "elasticsearch.thread_pool.queue",
            "elasticsearch.thread_pool.rejected",
            "elasticsearch.thread_pool.threads",
        }
        verify_custom(config, expected_metrics)


@pytest.mark.flaky(reruns=0)
def test_elasticsearch_with_enhanced_transport_stats():
    with run_service("elasticsearch/6.4.2", environment={"cluster.name": "testCluster"}) as es_container:
        host = container_ip(es_container)
        assert wait_for(
            p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180
        ), "service didn't start"
        config = dedent(
            f"""
            monitors:
            - type: elasticsearch
              host: {host}
              port: 9200
              username: elastic
              password: testing123
              enableEnhancedTransportStats: true
            """
        )
        expected_metrics = DEFAULT_METRICS | {
            "elasticsearch.transport.rx.count",
            "elasticsearch.transport.rx.size",
            "elasticsearch.transport.server_open",
            "elasticsearch.transport.tx.count",
            "elasticsearch.transport.tx.size",
        }
        verify_custom(config, expected_metrics)


# TODO: fix this failing test
# @pytest.mark.flaky(reruns=0)
# def test_elasticsearch_all_metrics():
#     with run_service("elasticsearch/6.4.2", environment={"cluster.name": "testCluster"}) as es_container:
#         host = container_ip(es_container)
#         assert wait_for(
#             p(http_status, url=f"http://{host}:9200/_all/_stats", status=[200]), 180
#         ), "service didn't start"
#         config = dedent(
#             f"""
#             monitors:
#             - type: elasticsearch
#               host: {host}
#               port: 9200
#               username: elastic
#               password: testing123
#               indexStatsMasterOnly: false
#               enableIndexStatsPrimaries: true
#               enableEnhancedClusterHealthStats: true
#               enableEnhancedHTTPStats: true
#               enableEnhancedJVMStats: true
#               enableEnhancedProcessStats: true
#               enableEnhancedThreadPoolStats: true
#               enableEnhancedTransportStats: true
#               enableEnhancedNodeIndicesStats:
#               - docs
#               - store
#               - indexing
#               - get
#               - search
#               - merges
#               - refresh
#               - flush
#               - warmer
#               - query_cache
#               - filter_cache
#               - fielddata
#               - completion
#               - segments
#               - translog
#               - request_cache
#               - recovery
#               - id_cache
#               - suggest
#               - percolate
#               enableEnhancedIndexStatsForIndexGroups:
#               - docs
#               - store
#               - indexing
#               - get
#               - search
#               - merges
#               - refresh
#               - flush
#               - warmer
#               - query_cache
#               - filter_cache
#               - fielddata
#               - completion
#               - segments
#               - translog
#               - request_cache
#               - recovery
#               - id_cache
#               - suggest
#               - percolate
#             """
#         )
#         with Agent.run(config) as agent:
#             want_metrics = METADATA.all_metrics
#             wait_for(lambda: len(agent.fake_services.datapoints) > 0)
#             got_metrics = frozenset([dp.metric for dp in agent.fake_services.datapoints])
#             assert want_metrics == got_metrics

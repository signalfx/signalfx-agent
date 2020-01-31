from contextlib import contextmanager
from functools import partial as p

import hashlib
import json
import pytest
import requests
import time

from datetime import datetime
from elasticsearch import Elasticsearch
from random import randint

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, http_status
from tests.helpers.util import container_ip, run_service, wait_for

VERSIONS = ["5.0.0", "7.5.1"]
AGENT_CONFIG_TEMPLATE = """
    monitors:
    - type: elasticsearch-query
      host: {host}
      port: 9200
      index: {index}
      elasticsearchRequest: '{query}'
    """

EXTENDED_STATS_METRICS = [
    "count",
    "min",
    "max",
    "avg",
    "sum",
    "sum_of_squares",
    "variance",
    "std_deviation",
    "std_deviation_bounds.lower",
    "std_deviation_bounds.upper",
]

PERCENTILE_METRICS = ["p1", "p5", "p25", "p50", "p75", "p95", "p99"]

HOSTS = ["nairobi", "helsniki", "madrid", "lisbon"]


def check_service_status(host):
    assert wait_for(p(http_status, url=f"http://{host}:9200/_nodes/_local", status=[200]), 180), "service didn't start"


@pytest.mark.parametrize("version", VERSIONS)
def test_elasticsearch_query_simple_metric_aggs(version):
    with run_service("elasticsearch/%s" % version) as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        write_data(host, version)

        query = {"query": {"match_all": {}}, "aggs": {"avg_cpu_utilization": {"avg": {"field": "cpu_utilization"}}}}

        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, index="metrics", query=json.dumps(query))

        with Agent.run(agent_config) as agent:
            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="elasticsearch_query.avg_cpu_utilization",
                    dimensions={"index": "metrics", "metric_aggregation_type": "avg"},
                )
            ), "Didn't get elasticsearch-query datapoints"


@pytest.mark.parametrize("version", VERSIONS)
def test_elasticsearch_query_extened_stats_aggs(version):

    with run_service("elasticsearch/%s" % version) as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        write_data(host, version)

        query = {
            "query": {"match_all": {}},
            "aggs": {"cpu_utilization_stats": {"extended_stats": {"field": "cpu_utilization"}}},
        }

        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, index="metrics", query=json.dumps(query))

        with Agent.run(agent_config) as agent:
            for metric in EXTENDED_STATS_METRICS:
                assert wait_for(
                    p(
                        has_datapoint,
                        agent.fake_services,
                        metric_name="elasticsearch_query.cpu_utilization_stats.%s" % metric,
                        dimensions={"index": "metrics", "metric_aggregation_type": "extended_stats"},
                    )
                ), "Didn't get elasticsearch-query datapoints"


@pytest.mark.parametrize("version", VERSIONS)
def test_elasticsearch_query_simple_metric_aggs_with_terms_aggs(version):
    with run_service("elasticsearch/%s" % version) as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        write_data(host, version)

        query = {
            "query": {"match_all": {}},
            "aggs": {
                "host_name": {
                    "terms": {"field": "host"},
                    "aggs": {"avg_cpu_utilization": {"avg": {"field": "cpu_utilization"}}},
                }
            },
        }

        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, index="metrics", query=json.dumps(query))

        with Agent.run(agent_config) as agent:
            for host in HOSTS:
                assert wait_for(
                    p(
                        has_datapoint,
                        agent.fake_services,
                        metric_name="elasticsearch_query.avg_cpu_utilization",
                        dimensions={"index": "metrics", "metric_aggregation_type": "avg", "host_name": host},
                    )
                ), "Didn't get elasticsearch-query datapoints"


@pytest.mark.parametrize("version", VERSIONS)
def test_elasticsearch_query_terminal_bucket_aggs(version):
    with run_service("elasticsearch/%s" % version) as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        write_data(host, version)

        query = {"query": {"match_all": {}}, "aggs": {"host_name": {"terms": {"field": "host"}}}}

        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, index="metrics", query=json.dumps(query))

        with Agent.run(agent_config) as agent:
            for host in HOSTS:
                assert wait_for(
                    p(
                        has_datapoint,
                        agent.fake_services,
                        metric_name="elasticsearch_query.host_name.doc_count",
                        dimensions={"index": "metrics", "bucket_aggregation_type": "terms", "host_name": host},
                    )
                ), "Didn't get elasticsearch-query datapoints"


@pytest.mark.parametrize("version", VERSIONS)
def test_elasticsearch_query_percentiles_aggs_with_terms_aggs(version):
    with run_service("elasticsearch/%s" % version) as es_container:
        host = container_ip(es_container)
        check_service_status(host)
        write_data(host, version)

        query = {
            "query": {"match_all": {}},
            "aggs": {
                "host_name": {
                    "terms": {"field": "host"},
                    "aggs": {"cpu_utilization_percentiles": {"percentiles": {"field": "cpu_utilization"}}},
                }
            },
        }

        agent_config = AGENT_CONFIG_TEMPLATE.format(host=host, index="metrics", query=json.dumps(query))

        with Agent.run(agent_config) as agent:
            for metric in PERCENTILE_METRICS:
                for host in HOSTS:
                    assert wait_for(
                        p(
                            has_datapoint,
                            agent.fake_services,
                            metric_name="elasticsearch_query.cpu_utilization_percentiles.%s" % metric,
                            dimensions={
                                "index": "metrics",
                                "metric_aggregation_type": "percentiles",
                                "host_name": host,
                            },
                        )
                    ), "Didn't get elasticsearch-query datapoints"


def write_data(host, version, num_docs=10):
    """
    Populates ES with mock data
    """
    es = Elasticsearch(hosts=[host])

    doc_type = "doc"
    mappings = {
        "mappings": {
            doc_type: {
                "properties": {
                    "host": {"type": "text", "fielddata": True},
                    "service": {"type": "text", "fielddata": True},
                    "container_id": {"type": "text", "fielddata": True},
                    "cpu_utilization": {"type": "double"},
                    "memory_utilization": {"type": "double"},
                    "@timestamp": {"type": "date"},
                }
            }
        }
    }

    if version.startswith("7"):
        doc_type = "_doc"
        mappings = {
            "mappings": {
                "properties": {
                    "host": {"type": "text", "fielddata": True},
                    "service": {"type": "text", "fielddata": True},
                    "container_id": {"type": "text", "fielddata": True},
                    "cpu_utilization": {"type": "double"},
                    "memory_utilization": {"type": "double"},
                    "@timestamp": {"type": "date"},
                }
            }
        }

    # create index with mappings
    es.indices.create(index="metrics", body=mappings, ignore=400)

    # metrics to mock
    metric_groups = ["cpu", "memory"]

    # dimensions to mock
    dimensions_set = [
        {"host": "nairobi", "service": "android", "container_id": "macbook"},
        {"host": "nairobi", "service": "ios", "container_id": "lenovo"},
        {"host": "helsniki", "service": "android", "container_id": "macbook"},
        {"host": "helsniki", "service": "ios", "container_id": "lenovo"},
        {"host": "madrid", "service": "android", "container_id": "macbook"},
        {"host": "madrid", "service": "ios", "container_id": "lenovo"},
        {"host": "lisbon", "service": "android", "container_id": "macbook"},
        {"host": "lisbon", "service": "ios", "container_id": "lenovo"},
    ]

    for i in range(num_docs):
        for dim_set in dimensions_set:
            id_str = ""
            doc = {}
            for mg in metric_groups:
                doc[mg + "_utilization"] = randint(0, 100)

            for dim_key, dim_val in dim_set.items():
                doc[dim_key] = dim_val
                id_str += dim_key + ":" + dim_val + "_"

            doc["@timestamp"] = i

            id_str += str(i)

            hash_object = hashlib.md5(id_str.encode("utf-8"))
            id = hash_object.hexdigest()
            res = es.index(index="metrics", doc_type=doc_type, id=id, body=doc)
            print("document created: %s" % doc)
            print(res)

        i = i + 1

from functools import partial as p
from pathlib import Path
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.util import container_ip, run_container, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.mongodb, pytest.mark.monitor_with_endpoints]

SCRIPT_DIR = Path(__file__).parent.resolve()


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

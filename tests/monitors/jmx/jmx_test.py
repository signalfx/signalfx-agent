from functools import partial as p
from pathlib import Path
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import run_service, wait_for

pytestmark = [pytest.mark.monitor_with_endpoints]


def test_jmx_monitor_basics():
    config = dedent(
        f"""
            observers:
             - type: docker
            monitors:
              - type: jmx
                intervalSeconds: 1
                username: cassandra
                password: cassandra
                discoveryRule: container_name == "cassandra-jmx-test" && port == 7199
                groovyScript: {{"#from": {Path(__file__).parent.resolve() / "cassandra.groovy"}, raw: true}}
            """
    )

    with run_service("cassandra", name="cassandra-jmx-test"):
        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint, agent.fake_services, metric_name="cassandra.state", count=5)
            ), "Didn't get datapoints"

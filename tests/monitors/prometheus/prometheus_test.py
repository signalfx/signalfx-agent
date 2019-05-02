from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import container_ip, ensure_always, run_service, wait_for

pytestmark = [pytest.mark.prometheus, pytest.mark.monitor_with_endpoints]


def test_prometheus_exporter():
    with run_service("dpgen", environment={"NUM_METRICS": 3}) as dpgen_cont:
        with Agent.run(
            dedent(
                f"""
                monitors:
                 - type: prometheus-exporter
                   host: {container_ip(dpgen_cont)}
                   port: 3000
                   intervalSeconds: 2
                   extraDimensions:
                     source: prometheus
                """
            )
        ) as agent:
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"source": "prometheus"})
            ), "didn't get prometheus datapoint"


def test_prometheus_exporter_basic_auth():
    # The dpgen service just checks that basic auth is present, not correct
    with run_service("dpgen", environment={"NUM_METRICS": 3, "REQUIRE_BASIC_AUTH": "yes"}) as dpgen_cont:
        with Agent.run(
            dedent(
                f"""
                monitors:
                 - type: prometheus-exporter
                   host: {container_ip(dpgen_cont)}
                   port: 3000
                   intervalSeconds: 2
                   extraDimensions:
                     source: prometheus
                """
            )
        ) as agent:
            assert ensure_always(
                lambda: not has_datapoint(agent.fake_services, dimensions={"source": "prometheus"}), timeout_seconds=5
            ), "got prometheus datapoint without basic auth (test setup is wrong)"

            agent.config["monitors"][0]["username"] = "bob"
            agent.config["monitors"][0]["password"] = "s3cr3t"
            agent.write_config()

            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"source": "prometheus"})
            ), "didn't get prometheus datapoint"

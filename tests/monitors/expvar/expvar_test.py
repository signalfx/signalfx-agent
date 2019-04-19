"""
Tests for the expvar monitor
"""
from functools import partial as p
from textwrap import dedent

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for

pytestmark = [pytest.mark.expvar, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("expvar")


def test_nginx():
    with run_service("expvar") as expvar_container:
        host = container_ip(expvar_container)
        assert wait_for(p(tcp_socket_open, host, 8080), 60), "service didn't start"

        with Agent.run(
            dedent(
                f"""
          monitors:
           - type: expvar
             host: {host}
             port: 8080
         """
            )
        ) as agent:
            for metric in METADATA.included_metrics:
                print("Waiting for %s" % metric)
                assert wait_for(
                    p(has_datapoint, agent.fake_services, metric_name=metric)
                ), "Didn't get included datapoints"

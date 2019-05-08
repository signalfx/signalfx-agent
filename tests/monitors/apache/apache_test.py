"""
Tests for the collectd/apache monitor
"""
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for
from tests.helpers.verify import verify

pytestmark = [pytest.mark.collectd, pytest.mark.apache, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("collectd/apache")


def run(config, metrics):
    with run_service("apache") as apache_container:
        host = container_ip(apache_container)
        config = config.format(host=host)
        assert wait_for(p(tcp_socket_open, host, 80), 60), "service didn't start"

        with Agent.run(config) as agent:
            verify(agent, metrics)
            assert has_datapoint_with_dim(agent.fake_services, "plugin", "apache"), "Didn't get apache datapoints"


def test_apache_included():
    run(
        """
        monitors:
        - type: collectd/apache
          host: {host}
          port: 80
        """,
        METADATA.included_metrics,
    )


def test_apache_all():
    run(
        """
        monitors:
        - type: collectd/apache
          host: {host}
          port: 80
          extraMetrics: ["*"]
        """,
        METADATA.all_metrics,
    )

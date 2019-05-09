"""
Tests for the collectd/nginx monitor
"""
from contextlib import contextmanager
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for
from tests.helpers.verify import verify

pytestmark = [pytest.mark.collectd, pytest.mark.nginx, pytest.mark.monitor_with_endpoints]


METADATA = Metadata.from_package("collectd/nginx")


@contextmanager
def run_nginx():
    with run_service("nginx") as nginx_container:
        host = container_ip(nginx_container)
        assert wait_for(p(tcp_socket_open, host, 80), 60), "service didn't start"
        yield host


def test_nginx_included():
    with run_nginx() as host, Agent.run(
        f"""
        monitors:
        - type: collectd/nginx
          host: {host}
          port: 80
        """
    ) as agent:
        verify(agent, METADATA.included_metrics)
        assert has_datapoint_with_dim(agent.fake_services, "plugin", "nginx"), "Didn't get nginx datapoints"


def test_nginx_all():
    with run_nginx() as host, Agent.run(
        f"""
        monitors:
        - type: collectd/nginx
          host: {host}
          port: 80
          extraMetrics: ["*"]
        """
    ) as agent:
        verify(agent, METADATA.all_metrics)
        assert has_datapoint_with_dim(agent.fake_services, "plugin", "nginx"), "Didn't get nginx datapoints"

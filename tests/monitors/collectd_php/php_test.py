"""
Tests for the collectd/php monitor
"""
from contextlib import contextmanager
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for
from tests.helpers.verify import verify

pytestmark = [pytest.mark.collectd, pytest.mark.php, pytest.mark.monitor_with_endpoints]


METADATA = Metadata.from_package("collectd/php")
INSTANCE = "test"


@contextmanager
def run_php_fpm():
    with run_service("php") as php_container:
        host = container_ip(php_container)
        assert wait_for(p(tcp_socket_open, host, 80), 60), "service didn't start"
        yield host


def test_php_default():
    with run_php_fpm() as host, Agent.run(
        f"""
        monitors:
        - type: collectd/php-fpm
          host: {host}
          port: 80
          name: {INSTANCE}
        """
    ) as agent:
        verify(agent, METADATA.default_metrics)
        assert has_datapoint_with_dim(agent.fake_services, "plugin", "curl_json"), "Didn't get php-fpm datapoints"
        assert has_datapoint_with_dim(
            agent.fake_services, "plugin_instance", INSTANCE
        ), "Didn't get right instance dimension on datapoints"

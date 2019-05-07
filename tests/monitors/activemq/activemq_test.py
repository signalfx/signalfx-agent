"""
Tests for the collectd/activemq monitor
"""
from functools import partial as p
from pathlib import Path

import pytest

from tests.helpers.assertions import tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for
from tests.helpers.verify import verify_included_metrics, verify_all_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.activemq, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("collectd/activemq")
SCRIPT_DIR = Path(__file__).parent.resolve()


def test_activemq_included():
    with run_service("activemq") as activemq_container:
        host = container_ip(activemq_container)
        config = f"""
        monitors:
          - type: collectd/activemq
            host: {host}
            port: 1099
            serviceURL: service:jmx:rmi:///jndi/rmi://{host}:1099/jmxrmi
            username: testuser
            password: testing123
        """

        assert wait_for(p(tcp_socket_open, host, 1099), 60), "service didn't start"
        verify_included_metrics(config, METADATA)


def test_activemq_all():
    with run_service("activemq") as activemq_container:
        host = container_ip(activemq_container)
        config = f"""
        monitors:
          - type: collectd/activemq
            host: {host}
            port: 1099
            serviceURL: service:jmx:rmi:///jndi/rmi://{host}:1099/jmxrmi
            username: testuser
            password: testing123
            extraMetrics: ["*"]
        """

        assert wait_for(p(tcp_socket_open, host, 1099), 60), "service didn't start"
        verify_all_metrics(config, METADATA)

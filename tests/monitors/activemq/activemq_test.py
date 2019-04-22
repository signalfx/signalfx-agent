"""
Tests for the collectd/activemq monitor
"""
from functools import partial as p
from pathlib import Path
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import any_metric_found, tcp_socket_open
from tests.helpers.util import container_ip, get_monitor_metrics_from_selfdescribe, run_service, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.activemq, pytest.mark.monitor_with_endpoints]

SCRIPT_DIR = Path(__file__).parent.resolve()


def test_activemq():
    with run_service("activemq") as activemq_container:
        host = container_ip(activemq_container)
        config = dedent(
            f"""
            monitors:
              - type: collectd/activemq
                host: {host}
                port: 1099
                serviceURL: service:jmx:rmi:///jndi/rmi://{host}:1099/jmxrmi
                username: testuser
                password: testing123
        """
        )
        assert wait_for(p(tcp_socket_open, host, 1099), 60), "service didn't start"
        with Agent.run(config) as agent:
            metrics = get_monitor_metrics_from_selfdescribe("collectd/activemq")
            assert wait_for(p(any_metric_found, agent.fake_services, metrics)), "Didn't get activemq datapoints"

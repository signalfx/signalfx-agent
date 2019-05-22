"""
Tests for the collectd/activemq monitor
"""
from contextlib import contextmanager
from functools import partial as p
from pathlib import Path

import pytest
from tests.helpers.assertions import tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for
from tests.helpers.verify import run_agent_verify_all_metrics, run_agent_verify_default_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.activemq, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("collectd/activemq")
SCRIPT_DIR = Path(__file__).parent.resolve()
JMX_USERNAME = "testuser"
JMX_PASSWORD = "testing123"


@contextmanager
def run_activemq():
    with run_service("activemq") as activemq_container:
        host = container_ip(activemq_container)
        assert wait_for(p(tcp_socket_open, host, 1099), 60), "broker socket didn't open"

        def check():
            # Check that the broker is actually responding. This currently isn't sufficient
            # to check that the broker is up for unknown reasons so still ignoring
            # error checking for now.
            res = activemq_container.exec_run(
                f"bin/activemq query --jmxuser {JMX_USERNAME} --jmxpassword {JMX_PASSWORD}"
            )
            return res.exit_code == 0 and b"Broker not available" not in res.output

        assert wait_for(check), "broker did not start"
        yield host


def test_activemq_default():
    with run_activemq() as host:
        config = f"""
        monitors:
        - type: collectd/activemq
          host: {host}
          port: 1099
          serviceURL: service:jmx:rmi:///jndi/rmi://{host}:1099/jmxrmi
          username: {JMX_USERNAME}
          password: {JMX_PASSWORD}
        """

        run_agent_verify_default_metrics(config, METADATA)


def test_activemq_all():
    with run_activemq() as host:
        config = f"""
        monitors:
        - type: collectd/activemq
          host: {host}
          port: 1099
          serviceURL: service:jmx:rmi:///jndi/rmi://{host}:1099/jmxrmi
          username: {JMX_USERNAME}
          password: {JMX_PASSWORD}
          extraMetrics: ["*"]
        """

        run_agent_verify_all_metrics(config, METADATA)

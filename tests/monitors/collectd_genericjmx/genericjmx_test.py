import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify_included_metrics
from tests.monitors.activemq.activemq_test import run_activemq, JMX_PASSWORD, JMX_USERNAME

pytestmark = [pytest.mark.collectd, pytest.mark.genericjmx, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("collectd/genericjmx")

# TODO: Test custom mbeans and other config options.


def test_genericjmx_included():
    with run_activemq() as host:
        run_agent_verify_included_metrics(
            f"""
            monitors:
            - type: collectd/genericjmx
              host: {host}
              port: 1099
              username: {JMX_USERNAME}
              password: {JMX_PASSWORD}
            """,
            METADATA,
        )

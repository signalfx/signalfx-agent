import pytest

from tests.helpers.assertions import has_log_message
from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify_all_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.uptime, pytest.mark.monitor_without_endpoints]


METADATA = Metadata.from_package("collectd/uptime")


def test_uptime():
    agent = run_agent_verify_all_metrics(
        """
        monitors:
        - type: collectd/uptime
        """,
        METADATA,
    )
    assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

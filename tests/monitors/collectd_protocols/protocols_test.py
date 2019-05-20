"""
Tests for the collectd/protocols monitor
"""

import pytest

from tests.helpers.assertions import has_log_message
from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify_included_metrics, run_agent_verify_all_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.protocols, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/protocols")


def test_protocols_included():
    """
    Test that we get all included metrics
    """
    agent = run_agent_verify_included_metrics(
        """
        monitors:
        - type: collectd/protocols
        """,
        METADATA,
    )
    assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_protocols_all():
    """
    Test that we get all metrics
    """
    agent = run_agent_verify_all_metrics(
        """
        monitors:
        - type: collectd/protocols
          extraMetrics: ["*"]
        """,
        METADATA,
    )
    assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

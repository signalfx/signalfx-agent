"""
Tests for the collectd/cpu monitor
"""
import pytest
from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify_all_metrics, run_agent_verify_default_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.cpu, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/cpu")


def test_collectd_cpu_default():
    run_agent_verify_default_metrics(
        """
        monitors:
        - type: collectd/cpu
        """,
        METADATA,
    )


def test_collectd_cpu_all():
    run_agent_verify_all_metrics(
        """
        monitors:
        - type: collectd/cpu
          extraMetrics: ["*"]
        """,
        METADATA,
    )

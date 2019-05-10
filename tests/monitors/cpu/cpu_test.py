"""
Tests for the cpu monitor
"""

import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify_included_metrics, run_agent_verify_all_metrics

pytestmark = [pytest.mark.windows, pytest.mark.cpu, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("cpu")


def test_cpu_included():
    run_agent_verify_included_metrics(
        """
        monitors:
        - type: cpu
        """,
        METADATA,
    )


def test_cpu_all():
    run_agent_verify_all_metrics(
        """
        monitors:
        - type: cpu
          extraMetrics: ["*"]
        """,
        METADATA,
    )

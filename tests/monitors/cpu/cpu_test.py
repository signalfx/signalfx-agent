"""
Tests for the cpu monitor
"""

import pytest
from tests.helpers.assertions import has_datapoint
from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify_all_metrics, run_agent_verify_default_metrics

pytestmark = [pytest.mark.windows, pytest.mark.cpu, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("cpu")


def test_cpu_default():
    run_agent_verify_default_metrics(
        """
        monitors:
        - type: cpu
        """,
        METADATA,
    )


def test_cpu_all():
    agent = run_agent_verify_all_metrics(
        """
        monitors:
        - type: cpu
          extraMetrics: ["*"]
        """,
        METADATA,
    )

    assert has_datapoint(agent.fake_services, metric_name="cpu.utilization_per_core", dimensions={"cpu": "0"})

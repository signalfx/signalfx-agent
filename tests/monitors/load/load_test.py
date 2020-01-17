"""
Tests for the cpu monitor
"""

import pytest
from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify_default_metrics

pytestmark = [pytest.mark.load, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("load")


def test_cpu_default():
    run_agent_verify_default_metrics(
        """
        monitors:
        - type: load
        """,
        METADATA,
    )

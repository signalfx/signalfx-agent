import sys

import pytest

from tests.helpers.agent import Agent
from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify

pytestmark = [pytest.mark.windows, pytest.mark.memory, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("memory")


def test_memory():
    expected_metrics = {"memory.used", "memory.utilization"}
    if sys.platform == "linux":
        expected_metrics.update(
            {"memory.buffered", "memory.cached", "memory.free", "memory.slab_recl", "memory.slab_unrecl"}
        )
    with Agent.run(
        """
        monitors:
          - type: memory
        """
    ) as agent:
        for met in expected_metrics:
            assert met in METADATA.included_metrics

        verify(agent, expected_metrics)

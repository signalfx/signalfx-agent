from functools import partial as p

import sys
import pytest

from helpers.assertions import has_any_metric_or_dim, has_log_message
from helpers.util import get_monitor_dims_from_selfdescribe, run_agent, wait_for

pytestmark = [pytest.mark.windows, pytest.mark.memory, pytest.mark.monitor_without_endpoints]


def test_memory():
    expected_metrics = ["memory.used", "memory.utilization"]
    if sys.platform == "linux":
        expected_metrics.extend(
            ["memory.buffered", "memory.cached", "memory.free", "memory.slab_recl", "memory.slab_unrecl"]
        )
    elif sys.platform == "win32" or sys.platform == "cygwin":
        expected_metrics.extend(["memory.available"])
    expected_dims = get_monitor_dims_from_selfdescribe("memory")
    with run_agent(
        """
    monitors:
      - type: memory
    """
    ) as [backend, get_output, _]:
        assert wait_for(
            p(has_any_metric_or_dim, backend, expected_metrics, expected_dims), timeout_seconds=60
        ), "timed out waiting for metrics and/or dimensions!"
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"

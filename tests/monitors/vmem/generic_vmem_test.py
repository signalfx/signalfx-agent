from functools import partial as p

import sys
import pytest

from helpers.assertions import has_any_metric_or_dim, has_log_message
from helpers.util import get_monitor_dims_from_selfdescribe, run_agent, wait_for

pytestmark = [pytest.mark.windows, pytest.mark.vmem, pytest.mark.monitor_without_endpoints]


def test_vmem():
    expected_metrics = []
    if sys.platform == "linux":
        expected_metrics.extend(
            [
                "vmpage_io.swap.in",
                "vmpage_io.swap.out",
                "vmpage_number.free_pages",
                "vmpage_number.mapped",
                "vmpage_io.memory.in",
                "vmpage_io.memory.out",
                "vmpage_faults.majflt",
                "vmpage_faults.minflt",
            ]
        )
    elif sys.platform == "win32" or sys.platform == "cygwin":
        expected_metrics.extend(
            ["vmpage.swap.in_per_second", "vmpage.swap.out_per_second", "vmpage.swap.total_per_second"]
        )
    expected_dims = get_monitor_dims_from_selfdescribe("vmem")
    with run_agent(
        """
    monitors:
      - type: vmem
    """
    ) as [backend, get_output, _]:
        assert wait_for(
            p(has_any_metric_or_dim, backend, expected_metrics, expected_dims), timeout_seconds=60
        ), "timed out waiting for metrics and/or dimensions!"
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"

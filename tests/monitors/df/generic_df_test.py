from functools import partial as p

import sys
import pytest

from helpers.assertions import has_any_metric_or_dim, has_log_message
from helpers.util import get_monitor_dims_from_selfdescribe, run_agent, wait_for

pytestmark = [pytest.mark.windows, pytest.mark.df, pytest.mark.monitor_without_endpoints]


def test_df():
    # TODO: make the helper that fetches metrics from selfdescribe.json check for platform specificity
    expected_metrics = [
        "df_complex.free",
        "df_complex.used",
        "percent_bytes.free",
        "percent_bytes.used",
        "disk.summary_utilization",
        "disk.utilization",
    ]
    if sys.platform == "linux":
        expected_metrics.extend(["df_inodes.free", "df_inodes.used", "percent_inodes.free", "percent_inodes.used"])
    expected_dims = get_monitor_dims_from_selfdescribe("df")
    with run_agent(
        """
    monitors:
      - type: df
    """
    ) as [backend, get_output, _]:
        assert wait_for(
            p(has_any_metric_or_dim, backend, expected_metrics, expected_dims), timeout_seconds=60
        ), "timed out waiting for metrics and/or dimensions!"
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"

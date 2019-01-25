from functools import partial as p
from textwrap import dedent
import pytest
import sys

from helpers.assertions import has_any_metric_or_dim, has_log_message
from helpers.util import get_monitor_dims_from_selfdescribe, get_monitor_metrics_from_selfdescribe, run_agent, wait_for


pytestmark = [
    pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows"),
    pytest.mark.windows,
    pytest.mark.windowslegacy,
]


def test_windowslegacy():
    expected_metrics = get_monitor_metrics_from_selfdescribe("windows-legacy")
    expected_dims = get_monitor_dims_from_selfdescribe("windows-legacy")
    config = dedent(
        """
        monitors:
         - type: windows-legacy
        """
    )
    with run_agent(config) as [backend, get_output, _]:
        assert wait_for(
            p(has_any_metric_or_dim, backend, expected_metrics, expected_dims), timeout_seconds=60
        ), "timed out waiting for metrics and/or dimensions!"
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"
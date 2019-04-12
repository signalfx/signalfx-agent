import sys
from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_any_metric_or_dim, has_log_message
from tests.helpers.util import get_monitor_dims_from_selfdescribe, get_monitor_metrics_from_selfdescribe, wait_for

pytestmark = [
    pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows"),
    pytest.mark.windows_only,
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
    with Agent.run(config) as agent:
        assert wait_for(
            p(has_any_metric_or_dim, agent.fake_services, expected_metrics, expected_dims), timeout_seconds=60
        ), "timed out waiting for metrics and/or dimensions!"
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

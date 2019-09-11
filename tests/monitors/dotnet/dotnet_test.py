import sys
from functools import partial as p
from textwrap import dedent

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_any_metric_or_dim, has_log_message
from tests.helpers.util import get_monitor_dims_from_selfdescribe, get_all_monitor_metrics_from_selfdescribe, wait_for

pytestmark = [
    pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows"),
    pytest.mark.windows_only,
    pytest.mark.dotnet,
]


def test_dotnet():
    expected_metrics = get_all_monitor_metrics_from_selfdescribe("dotnet")
    expected_dims = get_monitor_dims_from_selfdescribe("dotnet")
    config = dedent(
        """
        monitors:
         - type: dotnet
           extraMetrics:
             - net_clr_memory.num_total_reserved_bytes
             - net_clr_memory.num_bytes_in_all_heaps
             - net_clr_memory.num_gc_handles
        """
    )
    with Agent.run(config) as agent:
        assert wait_for(
            p(has_any_metric_or_dim, agent.fake_services, expected_metrics, expected_dims), timeout_seconds=60
        ), "timed out waiting for metrics and/or dimensions!"
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

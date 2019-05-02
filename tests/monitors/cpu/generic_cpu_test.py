"""
Tests for the cpu monitor
"""
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_any_metric_or_dim, has_log_message
from tests.helpers.util import get_monitor_dims_from_selfdescribe, get_monitor_metrics_from_selfdescribe, wait_for

pytestmark = [pytest.mark.windows, pytest.mark.cpu, pytest.mark.monitor_without_endpoints]


def test_cpu():
    expected_metrics = get_monitor_metrics_from_selfdescribe("cpu")
    expected_dims = get_monitor_dims_from_selfdescribe("cpu")
    with Agent.run(
        """
    monitors:
      - type: cpu
    """
    ) as agent:
        assert wait_for(
            p(has_any_metric_or_dim, agent.fake_services, expected_metrics, expected_dims), timeout_seconds=60
        ), "timed out waiting for metrics and/or dimensions!"
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

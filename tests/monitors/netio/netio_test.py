from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_any_metric_or_dim, has_log_message
from tests.helpers.util import (
    ensure_never,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    wait_for,
)

pytestmark = [pytest.mark.windows, pytest.mark.netio, pytest.mark.monitor_without_endpoints]


def test_netio():
    expected_metrics = get_monitor_metrics_from_selfdescribe("net-io")
    expected_dims = get_monitor_dims_from_selfdescribe("net-io")
    with Agent.run(
        """
    monitors:
      - type: net-io
    """
    ) as agent:
        assert wait_for(
            p(has_any_metric_or_dim, agent.fake_services, expected_metrics, expected_dims), timeout_seconds=60
        ), "timed out waiting for metrics and/or dimensions!"
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_netio_filter():
    expected_metrics = get_monitor_metrics_from_selfdescribe("net-io")
    try:
        expected_metrics.remove("network.total")
    except KeyError:
        pass

    with Agent.run(
        """
    procPath: /proc
    monitors:
      - type: net-io
        interfaces:
         - "!*"
    """
    ) as agent:
        assert wait_for(
            p(has_any_metric_or_dim, agent.fake_services, ["network.total"], []), timeout_seconds=60
        ), "timed out waiting for metrics and/or dimensions!"
        assert ensure_never(lambda: has_any_metric_or_dim(agent.fake_services, expected_metrics, []))
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

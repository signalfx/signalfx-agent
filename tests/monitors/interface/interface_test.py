import pytest

from tests.helpers.util import (
    get_monitor_metrics_from_selfdescribe,
    get_monitor_dims_from_selfdescribe,
    run_agent,
)
from tests.helpers.assertions import (
    has_any_metric_or_dim,
    has_log_message,
)

pytestmark = [pytest.mark.collectd, pytest.mark.interface, pytest.mark.monitor_without_endpoints]


def test_interface():
    expected_metrics = get_monitor_metrics_from_selfdescribe("collectd/interface")
    expected_dims = get_monitor_dims_from_selfdescribe("collectd/interface")
    with run_agent("""
    monitors:
      - type: collectd/interface
    """) as [backend, get_output, _]:
        assert has_any_metric_or_dim(backend, expected_metrics, expected_dims, timeout=60), \
            "timed out waiting for metrics and/or dimensions!"
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"

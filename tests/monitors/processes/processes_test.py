import os
from functools import partial as p

import pytest

from helpers.assertions import has_any_metric_or_dim, has_log_message
from helpers.util import run_agent, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.processes, pytest.mark.monitor_without_endpoints]


def test_processes():
    with open(os.path.join(os.path.dirname(os.path.realpath(__file__)), "metrics.txt"), "r") as fd:
        expected_metrics = {m.strip() for m in fd.readlines() if len(m.strip()) > 0}
    with run_agent(
        """
    procPath: /proc
    monitors:
      - type: collectd/processes
        collectContextSwitch: true
        processMatch:
          collectd: ".*collectd.*"
    """
    ) as [backend, get_output, _]:
        assert wait_for(
            p(has_any_metric_or_dim, backend, expected_metrics, None), timeout_seconds=60
        ), "timed out waiting for metrics and/or dimensions!"
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"

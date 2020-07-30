import sys
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, has_log_message
from tests.helpers.util import ensure_always, wait_for

pytestmark = [pytest.mark.windows, pytest.mark.diskio, pytest.mark.monitor_without_endpoints]


def test_diskio():
    # TODO: make the helper that fetches metrics from selfdescribe.json check for platform specificity
    expected_metrics = []
    if sys.platform == "linux":
        expected_metrics.extend(
            [
                "disk_merged.read",
                "disk_merged.write",
                "disk_octets.read",
                "disk_octets.write",
                "disk_ops.read",
                "disk_ops.write",
                "disk_ops.total",
                "disk_time.read",
                "disk_time.write",
                "disk_ops.pending",
            ]
        )
    elif sys.platform == "win32" or sys.platform == "cygwin":
        expected_metrics.extend(
            [
                "disk_ops.avg_read",
                "disk_ops.avg_write",
                "disk_octets.avg_read",
                "disk_octets.avg_write",
                "disk_time.avg_read",
                "disk_time.avg_write",
                "disk_ops.pending",
            ]
        )
    with Agent.run(
        """
    procPath: /proc
    monitors:
      - type: disk-io
        extraMetrics:
         - "*"
    """
    ) as agent:
        for metric in expected_metrics:
            assert wait_for(p(has_datapoint, agent.fake_services, metric_name=metric), timeout_seconds=20)
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_diskio_filter():
    with Agent.run(
        """
    procPath: /proc
    monitors:
      - type: disk-io
        intervalSeconds: 1
        disks:
         - "!*"
        datapointsToExclude:
         - metricName: disk_ops.total
    """
    ) as agent:
        assert ensure_always(lambda: not agent.fake_services.datapoints, timeout_seconds=7)
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

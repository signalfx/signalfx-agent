from functools import partial as p

import pytest

from helpers.assertions import has_datapoint, has_log_message
from helpers.util import ensure_always, run_agent, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.signalfx_metadata, pytest.mark.monitor_without_endpoints]


def test_signalfx_metadata():
    with run_agent(
        """
    procPath: /proc
    etcPath: /etc
    monitors:
      - type: collectd/signalfx-metadata
        persistencePath: /var/run/signalfx-agent
      - type: collectd/cpu
      - type: collectd/disk
      - type: collectd/memory
    """
    ) as [backend, get_output, _]:
        assert wait_for(p(has_datapoint, backend, "cpu.utilization", {"plugin": "signalfx-metadata"}))
        assert wait_for(p(has_datapoint, backend, "disk_ops.total", {"plugin": "signalfx-metadata"}))
        assert wait_for(p(has_datapoint, backend, "memory.utilization", {"plugin": "signalfx-metadata"}))
        assert ensure_always(
            lambda: not has_datapoint(backend, "cpu.utilization_per_core", {"plugin": "signalfx-metadata"})
        )
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"


def test_cpu_utilization_per_core():
    with run_agent(
        """
    monitors:
      - type: collectd/signalfx-metadata
        procFSPath: /proc
        etcPath: /etc
        persistencePath: /var/run/signalfx-agent
        perCoreCPUUtil: true
      - type: collectd/cpu
    metricsToInclude:
      - metricNames:
        - cpu.utilization_per_core
        monitorType: collectd/signalfx-metadata
        """
    ) as [backend, get_output, _]:
        assert wait_for(p(has_datapoint, backend, "cpu.utilization_per_core", {"plugin": "signalfx-metadata"}))
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"

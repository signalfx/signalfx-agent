import os

from tests.helpers import fake_backend
from tests.helpers.util import ensure_always, run_agent, wait_for
from tests.helpers.assertions import *

basic_config = """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/cpu
  - type: collectd/uptime
metricsToExclude:
  - metricName: cpu.utilization
"""

def test_basic_filtering():
    with run_agent(basic_config) as [backend, _]:
        wait_for(lambda: has_datapoint_with_metric_name(backend, "uptime"))
        ensure_always(lambda: not has_datapoint_with_metric_name(backend, "cpu.utilization"), 10)

negative_filtering_config = """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/memory
  - type: collectd/uptime
metricsToExclude:
  - metricName: memory.used
    negated: true
"""

def test_negated_filtering():
    with run_agent(negative_filtering_config) as [backend, _]:
        wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.used"))
        ensure_always(lambda: not has_datapoint_with_metric_name(backend, "uptime"), 10)

# Having monitorType in a filter should make that filter only apply to a
# specific monitor type and not to other metrics.
def test_negated_filter_with_monitor_type():
    with run_agent("""
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/memory
  - type: collectd/df
  - type: collectd/uptime
metricsToExclude:
  - metricNames:
     - memory.used
     - memory.free
    monitorType: collectd/memory
    negated: true
  - metricName: uptime
""") as [backend, _]:
        wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.used"))
        wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.free"))
        wait_for(lambda: has_datapoint_with_metric_name(backend, "df_complex.free"))
        ensure_always(lambda: not has_datapoint_with_metric_name(backend, "memory.cached"), 10)
        ensure_always(lambda: not has_datapoint_with_metric_name(backend, "uptime"), 5)

from helpers.assertions import has_datapoint_with_metric_name
from helpers.util import ensure_always, run_agent, wait_for

BASIC_CONFIG = """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/cpu
  - type: collectd/uptime
metricsToExclude:
  - metricName: cpu.utilization
"""


def test_basic_filtering():
    with run_agent(BASIC_CONFIG) as [backend, _, _]:
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "uptime"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(backend, "cpu.utilization"), 10)


NEGATIVE_FILTERING_CONFIG = """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/memory
  - type: collectd/uptime
metricsToExclude:
  - metricName: memory.used
    negated: true
"""


def test_negated_filtering():
    with run_agent(NEGATIVE_FILTERING_CONFIG) as [backend, _, _]:
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.used"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(backend, "uptime"), 10)


# Having monitorType in a filter should make that filter only apply to a
# specific monitor type and not to other metrics.
def test_negated_filter_with_monitor_type():
    with run_agent(
        """
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
"""
    ) as [backend, _, _]:
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.used"))
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.free"))
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "df_complex.free"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(backend, "memory.cached"), 10)
        assert ensure_always(lambda: not has_datapoint_with_metric_name(backend, "uptime"), 5)


def test_combined_filter_with_monitor_type():
    with run_agent(
        """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/memory
  - type: collectd/df
  - type: collectd/uptime
metricsToExclude:
  - metricNames:
     - memory.used
    monitorType: collectd/memory
    negated: true
  - metricName: uptime
  - metricNames:
    - memory.free
    monitorType: collectd/memory
    negated: true
"""
    ) as [backend, _, _]:
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.used"))
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.free"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(backend, "memory.cached"), 10)
        assert ensure_always(lambda: not has_datapoint_with_metric_name(backend, "uptime"), 5)


# Test overlapping filters with different negation. Blacklist is favored
def test_overlapping_filter_with_monitor_type():
    with run_agent(
        """
monitors:
  - type: collectd/memory
  - type: collectd/uptime
metricsToExclude:
  - metricName: uptime
    negated: true
    monitorType: collectd/uptime
  - metricName: uptime
    monitorType: collectd/uptime
"""
    ) as [backend, _, _]:
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.used"))
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.free"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(backend, "uptime"), 5)


# Test overlapping filters with different negation. Blacklist is favored
def test_overlapping_filter_with_monitor_type2():
    with run_agent(
        """
monitors:
  - type: collectd/memory
  - type: collectd/uptime
metricsToExclude:
  - metricName: uptime
    monitorType: collectd/uptime
  - metricName: uptime
    negated: true
    monitorType: collectd/uptime
"""
    ) as [backend, _, _]:
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.used"))
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.free"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(backend, "uptime"), 5)


# Ensure the filters get updated properly when the agent reloads a new config
def test_filter_with_restart():
    with run_agent(
        """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/df
  - type: collectd/memory
  - type: collectd/uptime
metricsToExclude:
  - metricNames:
     - memory.*
    monitorType: collectd/memory
"""
    ) as [backend, _, update_config]:
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "df_complex.free"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(backend, "memory.used"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(backend, "memory.free"))

        update_config(
            """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/df
  - type: collectd/memory
  - type: collectd/uptime
metricsToExclude:
  - metricNames:
     - memory.used
    monitorType: collectd/memory
"""
        )
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.free"))


# Ensure the filters on monitors get applied
def test_monitor_filter():
    with run_agent(
        """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/df
  - type: collectd/memory
    metricsToExclude:
     - metricName: memory.used
  - type: collectd/uptime
"""
    ) as [backend, _, update_config]:
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "df_complex.free"))
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.free"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(backend, "memory.used"))

        update_config(
            """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/df
  - type: collectd/memory
  - type: collectd/uptime
"""
        )
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.used"))
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.free"))


def test_mixed_regex_and_non_regex_filters():
    with run_agent(
        """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/memory
  - type: collectd/df
  - type: collectd/uptime
metricsToExclude:
  - metricNames:
     - /memory.used/
     - asdflkjassdf
    negated: true
"""
    ) as [backend, _, _]:
        assert wait_for(lambda: has_datapoint_with_metric_name(backend, "memory.used"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(backend, "memory.free"), 10)
        assert ensure_always(lambda: not has_datapoint_with_metric_name(backend, "uptime"), 5)

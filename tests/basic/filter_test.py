from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_metric_name
from tests.helpers.util import ensure_always, wait_for

BASIC_CONFIG = """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/cpu
  - type: collectd/uptime
metricsToExclude:
  - metricName: cpu.utilization
"""


def test_basic_filtering():
    with Agent.run(BASIC_CONFIG) as agent:
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "uptime"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(agent.fake_services, "cpu.utilization"), 10)


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
    with Agent.run(NEGATIVE_FILTERING_CONFIG) as agent:
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "memory.used"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(agent.fake_services, "uptime"), 10)


def test_negated_filter_with_monitor_type():
    """
    Having monitorType in a filter should make that filter only apply to a
    specific monitor type and not to other metrics.
    """
    with Agent.run(
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
    ) as agent:
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "memory.used"))
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "memory.free"))
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "df_complex.free"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(agent.fake_services, "memory.cached"), 10)
        assert ensure_always(lambda: not has_datapoint_with_metric_name(agent.fake_services, "uptime"), 5)


def test_combined_filter_with_monitor_type():
    with Agent.run(
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
    ) as agent:
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "memory.used"))
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "memory.free"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(agent.fake_services, "memory.cached"), 10)
        assert ensure_always(lambda: not has_datapoint_with_metric_name(agent.fake_services, "uptime"), 5)


def test_overlapping_filter_with_monitor_type():
    """
    Test overlapping filters with different negation. Blacklist is favored
    """
    with Agent.run(
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
    ) as agent:
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "memory.used"))
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "memory.free"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(agent.fake_services, "uptime"), 5)


def test_overlapping_filter_with_monitor_type2():
    """
    Test overlapping filters with different negation. Blacklist is favored
    """
    with Agent.run(
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
    ) as agent:
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "memory.used"))
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "memory.free"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(agent.fake_services, "uptime"), 5)


def test_include_filter_with_monitor_type():
    """
    Test that include filters will override exclude filters
    """
    with Agent.run(
        """
monitors:
  - type: collectd/disk
  - type: collectd/uptime
metricsToExclude:
  - metricNames:
    - disk_time.read
    monitorType: collectd/disk
  - metricNames:
    - disk_ops.read
    - disk_ops.write
    monitorType: collectd/disk
    negated: true
metricsToInclude:
  - metricNames:
    - disk_time.read
"""
    ) as agent:
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "disk_ops.read"))
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "disk_ops.write"), 5)
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "disk_time.read"), 5)
        assert ensure_always(lambda: not has_datapoint_with_metric_name(agent.fake_services, "disk_time.write"), 5)
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "uptime"), 5)


def test_filter_with_restart():
    """
    Ensure the filters get updated properly when the agent reloads a new config
    """
    with Agent.run(
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
    ) as agent:
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "df_complex.free"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(agent.fake_services, "memory.used"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(agent.fake_services, "memory.free"))

        agent.update_config(
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
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "memory.free"))


def test_monitor_filter():
    """
    Ensure the filters on monitors get applied
    """
    with Agent.run(
        """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/df
  - type: collectd/memory
    metricsToExclude:
     - metricName: memory.used
  - type: collectd/uptime
"""
    ) as agent:
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "df_complex.free"))
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "memory.free"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(agent.fake_services, "memory.used"))

        agent.update_config(
            """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/df
  - type: collectd/memory
  - type: collectd/uptime
"""
        )
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "memory.used"))
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "memory.free"))


def test_mixed_regex_and_non_regex_filters():
    with Agent.run(
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
    ) as agent:
        assert wait_for(lambda: has_datapoint_with_metric_name(agent.fake_services, "memory.used"))
        assert ensure_always(lambda: not has_datapoint_with_metric_name(agent.fake_services, "memory.free"), 10)
        assert ensure_always(lambda: not has_datapoint_with_metric_name(agent.fake_services, "uptime"), 5)

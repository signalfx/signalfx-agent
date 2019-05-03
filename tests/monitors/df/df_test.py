import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_log_message
from tests.helpers.metadata import Metadata
from tests.helpers.util import wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.df, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/df")


def test_df():
    with Agent.run(
        """
        monitors:
          - type: collectd/df
            hostFSPath: /
        """
    ) as agent:
        assert wait_for(lambda: len(agent.fake_services.datapoints_by_metric) > 0), "timed out waiting for metrics!"
        assert set(agent.fake_services.datapoints_by_metric) == METADATA.included_metrics
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_df_an_ungrouped_extra_metric():
    expected_metrics = METADATA.included_metrics | {"df_complex.reserved"}
    an_ungrouped_extra_metric = "df_complex.reserved"
    with Agent.run(
        f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - {an_ungrouped_extra_metric}
        """
    ) as agent:
        assert wait_for(lambda: len(agent.fake_services.datapoints_by_metric) > 0), "timed out waiting for metrics!"
        assert set(agent.fake_services.datapoints_by_metric) == expected_metrics
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_df_a_grouped_extra_metric1():
    expected_metrics = METADATA.included_metrics | {"df_inodes.free", "df_inodes.reserved", "df_inodes.used"}
    a_grouped_extra_metric = "df_inodes.reserved"
    with Agent.run(
            f"""
            monitors:
              - type: collectd/df
                hostFSPath: /
                extraMetrics:
                - {a_grouped_extra_metric}
            """
    ) as agent:
        assert wait_for(lambda: len(agent.fake_services.datapoints_by_metric) > 0), "timed out waiting for metrics!"
        assert set(agent.fake_services.datapoints_by_metric) == expected_metrics
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_df_a_grouped_extra_metric2():
    expected_metrics = METADATA.included_metrics | \
                       {"percent_bytes.free", "percent_bytes.reserved", "percent_bytes.used"}
    a_grouped_extra_metric = "percent_bytes.used"
    with Agent.run(
            f"""
            monitors:
              - type: collectd/df
                hostFSPath: /
                extraMetrics:
                - {a_grouped_extra_metric}
            """
    ) as agent:
        assert wait_for(lambda: len(agent.fake_services.datapoints_by_metric) > 0), "timed out waiting for metrics!"
        assert set(agent.fake_services.datapoints_by_metric) == expected_metrics
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_df_extra_metrics_all():
    expected_metrics = METADATA.all_metrics
    with Agent.run(
        """
        monitors:
          - type: collectd/df
            hostFSPath: /
            valuesPercentage: true
            reportInodes: true
            extraMetrics:
            - df_*
            - percent_*
        """
    ) as agent:
        assert wait_for(lambda: len(agent.fake_services.datapoints_by_metric) > 0), "timed out waiting for metrics!"
        assert set(agent.fake_services.datapoints_by_metric) == expected_metrics
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

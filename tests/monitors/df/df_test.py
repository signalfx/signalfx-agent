import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_log_message
from tests.helpers.metadata import Metadata
from tests.helpers.util import wait_for
from tests.helpers.verify import verify_custom

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
        assert wait_for(lambda: agent.fake_services.datapoints_by_metric), "timed out waiting for metrics!"
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
        assert wait_for(lambda: agent.fake_services.datapoints_by_metric), "timed out waiting for metrics!"
        assert set(agent.fake_services.datapoints_by_metric) == expected_metrics
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_df_a_grouped_extra_metric1():
    verify_custom(
        """
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - df_inodes.reserved
        """,
        METADATA.included_metrics | {"df_inodes.reserved"},
    )


def test_df_a_grouped_extra_metric2():
    verify_custom(
        """
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - percent_bytes.used
            - percent_bytes.free
        """,
        METADATA.included_metrics | {"percent_bytes.used", "percent_bytes.free"},
    )


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
        assert wait_for(lambda: agent.fake_services.datapoints_by_metric), "timed out waiting for metrics!"
        assert set(agent.fake_services.datapoints_by_metric) == expected_metrics
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

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
        enableExtraMetricsFilter: true
        monitors:
          - type: collectd/df
            hostFSPath: /
        """
    ) as agent:
        _ = (
            wait_for(lambda: set(agent.fake_services.datapoints_by_metric) == METADATA.included_metrics),
            "timed out waiting for metrics and/or dimensions!",
        )
        assert set(agent.fake_services.datapoints_by_metric) == METADATA.included_metrics
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_df_extra_metrics():
    expected_metrics = METADATA.included_metrics | {"df_complex.reserved"}
    with Agent.run(
        """
        enableExtraMetricsFilter: true
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - df_complex.reserved
        """
    ) as agent:
        _ = (
            wait_for(lambda: set(agent.fake_services.datapoints_by_metric) == expected_metrics),
            "timed out waiting for metrics and/or dimensions!",
        )
        assert set(agent.fake_services.datapoints_by_metric) == expected_metrics
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_df_extra_metrics_all():
    expected_metrics = METADATA.all_metrics
    with Agent.run(
        """
        enableExtraMetricsFilter: true
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
        _ = (
            wait_for(lambda: set(agent.fake_services.datapoints_by_metric) == expected_metrics),
            "timed out waiting for metrics and/or dimensions!",
        )
        assert set(agent.fake_services.datapoints_by_metric) == expected_metrics
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

import pytest

from tests.helpers.assertions import has_log_message
from tests.helpers.metadata import get_metadata
from tests.helpers.util import run_agent, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.df, pytest.mark.monitor_without_endpoints]

METADATA = get_metadata("collectd/df")


def test_df():
    with run_agent(
        """
        monitors:
          - type: collectd/df
            hostFSPath: /
        """
    ) as (backend, get_output, _):
        _ = (
            wait_for(lambda: set(backend.datapoints_by_metric) == METADATA.included_metrics),
            "timed out waiting for metrics and/or dimensions!",
        )
        assert set(backend.datapoints_by_metric) == METADATA.included_metrics
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"


def test_df_additional_metrics():
    expected_metrics = METADATA.included_metrics | {"df_complex.reserved"}
    with run_agent(
        """
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - df_complex.reserved
        """
    ) as (backend, get_output, _):
        _ = (
            wait_for(lambda: set(backend.datapoints_by_metric) == expected_metrics),
            "timed out waiting for metrics and/or dimensions!",
        )
        assert set(backend.datapoints_by_metric) == expected_metrics
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"


def test_df_additional_metrics_all():
    expected_metrics = METADATA.all_metrics
    with run_agent(
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
    ) as (backend, get_output, _):
        _ = (
            wait_for(lambda: set(backend.datapoints_by_metric) == expected_metrics),
            "timed out waiting for metrics and/or dimensions!",
        )
        assert set(backend.datapoints_by_metric) == expected_metrics
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"

import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify_custom, verify_included_metrics, verify_all_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.df, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/df")


def test_df_included_metrics():
    agent_config = """
        monitors:
          - type: collectd/df
            hostFSPath: /
        """
    verify_included_metrics(agent_config, METADATA)


def test_df_extra_metrics():
    df_complex_reserved, df_inodes_reserved = "df_complex.reserved", "df_inodes.reserved"
    expected_metrics = METADATA.included_metrics | {df_complex_reserved, df_inodes_reserved}
    agent_config = f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - {df_complex_reserved}
            - {df_inodes_reserved}
        """
    verify_custom(agent_config, expected_metrics)


def test_df_report_inodes_flag():
    expected_metrics = METADATA.included_metrics | METADATA.metrics_by_group["report-inodes"]
    agent_config = f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            reportInodes: true
        """
    verify_custom(agent_config, expected_metrics)


def test_df_values_percentage_flag():
    expected_metrics = METADATA.included_metrics | METADATA.metrics_by_group["values-percentage"]
    agent_config = f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            valuesPercentage: true
        """
    verify_custom(agent_config, expected_metrics)


def test_df_report_inodes_and_values_percentage_flags():
    expected_metrics = METADATA.all_metrics - {"df_complex.reserved"}
    agent_config = f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            reportInodes: true
            valuesPercentage: true
        """
    verify_custom(agent_config, expected_metrics)


def test_df_extra_metrics_all():
    agent_config = f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - '*'
        """
    verify_all_metrics(agent_config, METADATA)

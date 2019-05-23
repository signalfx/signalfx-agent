import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify, run_agent_verify_default_metrics, run_agent_verify_all_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.df, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/df")


def test_df_default_metrics():
    agent_config = """
        monitors:
          - type: collectd/df
        """
    run_agent_verify_default_metrics(agent_config, METADATA)


def test_df_extra_metrics():
    df_complex_reserved, df_inodes_reserved = "df_complex.reserved", "df_inodes.reserved"
    expected_metrics = METADATA.default_metrics | {df_complex_reserved, df_inodes_reserved}
    agent_config = f"""
        monitors:
          - type: collectd/df
            extraMetrics:
            - {df_complex_reserved}
            - {df_inodes_reserved}
        """
    run_agent_verify(agent_config, expected_metrics)


def test_df_inodes_flag():
    expected_metrics = METADATA.default_metrics | METADATA.metrics_by_group["inodes"]
    agent_config = f"""
        monitors:
          - type: collectd/df
            reportInodes: true
        """
    run_agent_verify(agent_config, expected_metrics)


def test_df_percentage_flag():
    expected_metrics = METADATA.default_metrics | METADATA.metrics_by_group["percentage"]
    agent_config = f"""
        monitors:
          - type: collectd/df
            valuesPercentage: true
        """
    run_agent_verify(agent_config, expected_metrics)


def test_df_inodes_and_percentage_flags():
    expected_metrics = METADATA.all_metrics - {"df_complex.reserved"}
    agent_config = f"""
        monitors:
          - type: collectd/df
            reportInodes: true
            valuesPercentage: true
        """
    run_agent_verify(agent_config, expected_metrics)


def test_df_extra_metrics_all():
    agent_config = f"""
        monitors:
          - type: collectd/df
            extraMetrics:
            - '*'
        """
    run_agent_verify_all_metrics(agent_config, METADATA)

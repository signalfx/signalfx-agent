import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify_custom

pytestmark = [pytest.mark.collectd, pytest.mark.df, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/df")


def test_df_included_metrics():
    expected_metrics = METADATA.included_metrics
    agent_config = """
        monitors:
          - type: collectd/df
            hostFSPath: /
        """
    verify_custom(agent_config, expected_metrics)


def test_df_extra_metrics():
    extra_metric_config_1, extra_metric_config_2 = "df_complex.reserved", "df_inodes.reserved"
    expected_metrics = METADATA.included_metrics | {extra_metric_config_1, extra_metric_config_2}
    agent_config = f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - {extra_metric_config_1}
            - {extra_metric_config_2}
        """
    verify_custom(agent_config, expected_metrics)


def test_df_extra_metrics_by_wildcard():
    percent_prefixed_wildcard_extra_metric_config = "percent_*"
    expected_metrics = METADATA.included_metrics | {
        "percent_bytes.free",
        "percent_bytes.reserved",
        "percent_bytes.used",
        "percent_inodes.free",
        "percent_inodes.reserved",
        "percent_inodes.used",
    }
    agent_config = f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - {percent_prefixed_wildcard_extra_metric_config}
        """
    verify_custom(agent_config, expected_metrics)


def test_df_invalid_extra_metric():
    expected_metrics = METADATA.included_metrics
    invalid_extra_metric_config = "Y2W8OBrdZZ"
    agent_config = f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - {invalid_extra_metric_config}
        """
    verify_custom(agent_config, expected_metrics)


def test_df_extra_metrics_all():
    expected_metrics = METADATA.all_metrics
    any_wildcard_in_quotes_extra_metric_config = "'*'"
    agent_config = f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - {any_wildcard_in_quotes_extra_metric_config}
        """
    verify_custom(agent_config, expected_metrics)

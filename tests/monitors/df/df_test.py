import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify_custom

pytestmark = [pytest.mark.collectd, pytest.mark.df, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/df")


def test_df_included_metrics():
    expected_metrics = METADATA.included_metrics
    verify_custom(
        """
        monitors:
          - type: collectd/df
            hostFSPath: /
        """,
        expected_metrics,
    )


def test_df_extra_metrics():
    extra_metric_1, extra_metric_2 = "df_complex.reserved", "df_inodes.reserved"
    expected_metrics = METADATA.included_metrics | {extra_metric_1, extra_metric_2}
    verify_custom(
        f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - {extra_metric_1}
            - {extra_metric_2}
        """,
        expected_metrics,
    )


def test_df_percent_prefixed_metrics():
    expected_metrics = METADATA.included_metrics | {
        "percent_bytes.free",
        "percent_bytes.reserved",
        "percent_bytes.used",
        "percent_inodes.free",
        "percent_inodes.reserved",
        "percent_inodes.used",
    }
    percent_prefixed_wildcard = "percent_*"
    verify_custom(
        f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - {percent_prefixed_wildcard}
        """,
        expected_metrics,
    )


def test_df_invalid_extra_metric():
    expected_metrics = METADATA.included_metrics
    invalid_extra_metric = "Y2W8OBrdZZ"
    verify_custom(
        f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - {invalid_extra_metric}
        """,
        expected_metrics,
    )


# TODO: Fix failing test
@pytest.mark.skip(reason="failing due to bug")
def test_df_blank_extra_metric():
    expected_metrics = METADATA.included_metrics
    blank_extra_metric = " "
    verify_custom(
        f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - {blank_extra_metric}
        """,
        expected_metrics,
    )


def test_df_extra_metrics_all():
    expected_metrics = METADATA.all_metrics
    any_wildcard_in_quotes = "'*'"
    verify_custom(
        f"""
        monitors:
          - type: collectd/df
            hostFSPath: /
            extraMetrics:
            - {any_wildcard_in_quotes}
        """,
        expected_metrics,
    )

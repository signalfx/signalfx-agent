"""
Tests for the expvar monitor
"""

import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify_included_metrics, verify_all_metrics

pytestmark = [pytest.mark.expvar, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("expvar")


def test_expvar_included(expvar):
    verify_included_metrics(
        f"""
        enableExtraMetricsFilter: true
        monitors:
        - type: expvar
          host: {expvar}
          port: 8080
        """,
        METADATA,
    )


def test_expvar_enhanced(expvar):
    verify_all_metrics(
        f"""
        enableExtraMetricsFilter: true
        monitors:
        - type: expvar
          host: {expvar}
          port: 8080
          enhancedMetrics: true
        """,
        METADATA,
    )


# TODO: test manually specified metric paths

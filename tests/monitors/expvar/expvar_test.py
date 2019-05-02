"""
Tests for the expvar monitor
"""

import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify_included_metrics, verify_all_metrics, verify_custom

pytestmark = [pytest.mark.expvar, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("expvar")


def test_expvar_included(expvar_container_ip):
    verify_included_metrics(
        f"""
        monitors:
        - type: expvar
          host: {expvar_container_ip}
          port: 8080
        """,
        METADATA,
    )


def test_expvar_enhanced(expvar_container_ip):
    verify_all_metrics(
        f"""
        monitors:
        - type: expvar
          host: {expvar_container_ip}
          port: 8080
          enhancedMetrics: true
        """,
        METADATA,
    )


def test_expvar_custom_metric(expvar_container_ip):
    expected = METADATA.included_metrics | {"queues.count"}
    verify_custom(
        f"""
        monitors:
        - type: expvar
          host: {expvar_container_ip}
          port: 8080
          metrics:
          - JSONPath: queues.count
            type: gauge
        """,
        expected,
    )

import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify_included_metrics, verify_custom

pytestmark = [pytest.mark.collectd, pytest.mark.disk, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/disk")

# Didn't show up locally but did in CI. Maybe only reported when non-zero?
EXCLUDED = {"pending_operations"}


def test_disk_included():
    verify_included_metrics(
        """
        monitors:
        - type: collectd/disk
        """,
        METADATA,
    )


def test_disk_all():
    verify_custom(
        """
        monitors:
        - type: collectd/disk
          extraMetrics: ["*"]
        """,
        METADATA.all_metrics - EXCLUDED,
    )

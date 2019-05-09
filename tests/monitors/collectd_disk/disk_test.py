import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify_included_metrics, verify_all_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.disk, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/disk")


def test_disk_included():
    verify_included_metrics(
        """
        monitors:
        - type: collectd/disk
        """,
        METADATA,
    )


def test_disk_all():
    verify_all_metrics(
        """
        monitors:
        - type: collectd/disk
          extraMetrics: ["*"]
        """,
        METADATA,
    )

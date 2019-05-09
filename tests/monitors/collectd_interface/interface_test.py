import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify_included_metrics, verify_all_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.interface, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/netinterface")


def test_interface_included():
    verify_included_metrics(
        """
        monitors:
        - type: collectd/interface
        """,
        METADATA,
    )


def test_interface_all():
    verify_all_metrics(
        """
        monitors:
        - type: collectd/interface
          extraMetrics: ["*"]
        """,
        METADATA,
    )

import pytest

from tests.helpers.agent import Agent
from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify_included_metrics, verify_expected_is_subset

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
    with Agent.run(
        """
        monitors:
        - type: collectd/disk
          extraMetrics: ["*"]
        """
    ) as agent:
        # pending_operations only shows up sometimes on CI. Maybe only reported when non-zero?
        verify_expected_is_subset(agent, METADATA.all_metrics - {"pending_operations"})

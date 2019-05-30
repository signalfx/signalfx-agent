import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify_default_metrics, run_agent_verify_all_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.interface, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/netinterface")


def test_interface_default():
    run_agent_verify_default_metrics(
        """
        monitors:
        - type: collectd/interface
        """,
        METADATA,
    )


def test_interface_all():
    run_agent_verify_all_metrics(
        """
        monitors:
        - type: collectd/interface
          extraMetrics: ["*"]
        """,
        METADATA,
    )

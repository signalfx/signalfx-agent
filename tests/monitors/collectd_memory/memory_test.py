import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify_default_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.memory, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/memory")


def test_collectd_memory_default():
    run_agent_verify_default_metrics(
        """
        monitors:
        - type: collectd/memory
        """,
        METADATA,
    )


# Only has default metrics so no test for all metrics.

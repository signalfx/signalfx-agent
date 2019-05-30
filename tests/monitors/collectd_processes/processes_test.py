import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify_all_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.processes, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/processes")


def test_processes_all():
    run_agent_verify_all_metrics(
        """
        procPath: /proc
        monitors:
          - type: collectd/processes
            collectContextSwitch: true
            processMatch:
              collectd: ".*collectd.*"
        """,
        METADATA,
    )

import sys

import pytest

from tests.helpers.assertions import has_log_message
from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify

pytestmark = [pytest.mark.windows, pytest.mark.vmem, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("vmem")
METRICS = set()

if sys.platform == "linux":
    METRICS.update(
        {
            "vmpage_io.swap.in",
            "vmpage_io.swap.out",
            "vmpage_number.free_pages",
            "vmpage_number.mapped",
            "vmpage_io.memory.in",
            "vmpage_io.memory.out",
            "vmpage_faults.majflt",
            "vmpage_faults.minflt",
            "vmpage_number.shmem_pmdmapped",
        }
    )
elif sys.platform == "win32" or sys.platform == "cygwin":
    METRICS.update({"vmpage.swap.in_per_second", "vmpage.swap.out_per_second", "vmpage.swap.total_per_second"})


def test_vmem_default():
    agent = run_agent_verify(
        """
        monitors:
        - type: vmem
        """,
        METRICS & METADATA.default_metrics,
    )
    assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_vmem_all():
    agent = run_agent_verify(
        """
        monitors:
        - type: vmem
          extraMetrics: ["*"]
        """,
        METRICS,
    )
    assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

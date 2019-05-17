import pytest

from tests.helpers.assertions import has_log_message
from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify_included_metrics, run_agent_verify_all_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.vmem, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/vmem")


def test_collectd_vmem_included():
    agent = run_agent_verify_included_metrics(
        """
        monitors:
        - type: collectd/vmem
        """,
        METADATA,
    )
    assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_collectd_vmem_all():
    agent = run_agent_verify_all_metrics(
        """
        monitors:
        - type: collectd/vmem
          extraMetrics: ["*"]
        """,
        METADATA,
    )
    assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

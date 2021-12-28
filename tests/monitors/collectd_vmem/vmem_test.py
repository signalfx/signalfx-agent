import pytest

from tests.helpers.assertions import has_log_message
from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify_default_metrics, verify_expected_is_subset
from tests.helpers.agent import Agent

pytestmark = [pytest.mark.collectd, pytest.mark.vmem, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/vmem")


def test_collectd_vmem_default():
    agent = run_agent_verify_default_metrics(
        """
        monitors:
        - type: collectd/vmem
        """,
        METADATA,
    )
    assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_collectd_vmem_all():
    metrics = METADATA.all_metrics
    with Agent.run(
        f"""
        monitors:
        - type: collectd/vmem
          extraMetrics: ["*"]
        """
    ) as agent:
        verify_expected_is_subset(agent, metrics)

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_log_message
from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify

pytestmark = [pytest.mark.collectd, pytest.mark.load, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/load")


def test_load_default():
    with Agent.run(
        """
        monitors:
        - type: collectd/load
        """
    ) as agent:
        verify(agent, METADATA.default_metrics)
    assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

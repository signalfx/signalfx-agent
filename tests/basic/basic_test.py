"""
Very basic tests of the agent
"""

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_log_message
from tests.helpers.util import wait_for


def test_basic():
    """
    See if we get datapoints from a very standard set of monitors
    """
    with Agent.run(
        """
monitors:
  - type: cpu
  - type: collectd/uptime
"""
    ) as agent:
        assert wait_for(lambda: agent.fake_services.datapoints), "Didn't get any datapoints"
        assert has_log_message(agent.output, "info")

"""
Very basic tests of the agent
"""

from tests.helpers.assertions import has_log_message
from tests.helpers.util import run_agent, wait_for

BASIC_CONFIG = """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/cpu
  - type: collectd/uptime
"""


def test_basic():
    """
    See if we get datapoints from a very standard set of monitors
    """
    with run_agent(BASIC_CONFIG) as [backend, get_output, _]:
        assert wait_for(lambda: backend.datapoints), "Didn't get any datapoints"
        assert has_log_message(get_output(), "info")

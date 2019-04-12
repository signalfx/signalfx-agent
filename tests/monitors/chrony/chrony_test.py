"""
Integration tests for chrony monitor.
"""
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_log_message
from tests.helpers.util import wait_for

CHRONY_CONFIG = """
monitors:
  - type: collectd/chrony
    host: localhost
    port: 23874
"""


def test_chrony():
    """
    Unfortunately, chronyd is very hard to run in a test environment without
    giving it the ability to change the time which we don't want, so just check
    for an error message ensuring that the monitor actually did configure it,
    even if it doesn't emit any metrics.
    """
    with Agent.run(CHRONY_CONFIG) as agent:

        def has_error():
            return has_log_message(
                agent.output, level="error", message="chrony plugin: chrony_query (REQ_TRACKING) failed"
            )

        assert wait_for(has_error), "Didn't get chrony error message"

import time

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_log_message

pytestmark = [pytest.mark.collectd, pytest.mark.cpufreq, pytest.mark.monitor_without_endpoints]


def test_cpufreq():
    with Agent.run(
        """
    monitors:
      - type: collectd/cpufreq
    """
    ) as agent:
        time.sleep(10)
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

import time

import pytest

from tests.helpers.assertions import has_log_message
from tests.helpers.util import run_agent

pytestmark = [pytest.mark.collectd, pytest.mark.cpufreq, pytest.mark.monitor_without_endpoints]


def test_cpufreq():
    with run_agent(
        """
    monitors:
      - type: collectd/cpufreq
    """
    ) as [_, get_output, _]:
        time.sleep(10)
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"

from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.util import wait_for

pytestmark = [pytest.mark.windows_only, pytest.mark.windows, pytest.mark.win_services, pytest.mark.telegraf]

MONITOR_CONFIG = """
monitors:
  - type: telegraf/win_services
"""


def test_win_services():
    with Agent.run(MONITOR_CONFIG) as agent:
        assert wait_for(
            p(has_datapoint_with_dim, agent.fake_services, "plugin", "telegraf-win_services")
        ), "didn't get datapoint with expected plugin dimension"

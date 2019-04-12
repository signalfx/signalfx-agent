from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, has_datapoint_with_metric_name
from tests.helpers.util import wait_for

pytestmark = [pytest.mark.windows, pytest.mark.telegraf_procstat, pytest.mark.telegraf]

MONITOR_CONFIG = """
monitors:
  - type: telegraf/procstat
    exe: "signalfx-agent*"
  - type: telegraf/procstat
    exe: "SIGNALFX-AGENT*"
"""


def test_telegraf_procstat():
    with Agent.run(MONITOR_CONFIG) as agent:
        # wait for fake ingest to receive the procstat metrics
        assert wait_for(
            p(has_datapoint_with_metric_name, agent.fake_services, "procstat.cpu_usage")
        ), "no cpu usage datapoint found for process"
        assert wait_for(
            p(has_datapoint_with_dim, agent.fake_services, "plugin", "telegraf-procstat")
        ), "plugin dimension not set"

from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_metric_name
from tests.helpers.util import wait_for

pytestmark = [pytest.mark.internal_metrics, pytest.mark.monitor_without_endpoints]


CONFIG = """
monitors:
  - type: internal-metrics

"""


def test_internal_metrics():
    with Agent.run(CONFIG) as agent:
        assert wait_for(
            p(has_datapoint_with_metric_name, agent.fake_services, "sfxagent.datapoints_sent")
        ), "Didn't get internal metric datapoints"

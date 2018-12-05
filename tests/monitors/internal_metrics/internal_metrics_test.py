from functools import partial as p

import pytest

from helpers.assertions import has_datapoint_with_metric_name
from helpers.util import run_agent, wait_for

pytestmark = [pytest.mark.internal_metrics, pytest.mark.monitor_without_endpoints]


CONFIG = """
monitors:
  - type: internal-metrics

"""


def test_internal_metrics():
    with run_agent(CONFIG) as [backend, _, _]:
        assert wait_for(
            p(has_datapoint_with_metric_name, backend, "sfxagent.datapoints_sent")
        ), "Didn't get internal metric datapoints"

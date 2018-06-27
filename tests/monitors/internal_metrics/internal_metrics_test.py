from functools import partial as p
import pytest

from tests.helpers.util import wait_for, run_agent
from tests.helpers.assertions import *

pytestmark = [pytest.mark.internal_metrics, pytest.mark.monitor_without_endpoints]


config = """
monitors:
  - type: internal-metrics

"""

def test_internal_metrics():
    with run_agent(config) as [backend, _, _]:
        assert wait_for(p(has_datapoint_with_metric_name, backend, "sfxagent.datapoints_sent")), "Didn't get internal metric datapoints"

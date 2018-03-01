from functools import partial as p
import os
import string

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, run_agent
from tests.helpers.assertions import *

config = """
monitors:
  - type: internal-metrics

"""

def test_internal_metrics():
    with run_agent(config) as [backend, _]:
        assert wait_for(p(has_datapoint_with_metric_name, backend, "sfxagent.datapoints_sent")), "Didn't get internal metric datapoints"

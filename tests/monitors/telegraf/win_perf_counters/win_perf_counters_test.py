from functools import partial as p
import os
import pytest
import string
import sys

from tests.helpers.util import wait_for, run_agent, run_service, container_ip
from tests.helpers.assertions import *

pytestmark = [pytest.mark.windows, pytest.mark.telegraf, pytest.mark.win_perf_counters]

config = """
monitors:
 - type: telegraf/win_perf_counters
   printValid: true
   objects:
    - objectName: "Processor"
      instances:
       - "*"
      counters:
       - "% Idle Time"
       - "% Interrupt Time"
       - "% Privileged Time"
       - "% User Time"
       - "% Processor Time"
      includeTotal: true
      measurement: "win_cpu"
"""

metrics = [
    "win_cpu.Percent_Idle_Time",
    "win_cpu.Percent_Interrupt_Time",
    "win_cpu.Percent_Privileged_Time",
    "win_cpu.Percent_Processor_Time",
    "win_cpu.Percent_User_Time",
]

@pytest.mark.skipif(sys.platform != 'win32', reason="only runs on windows")
def test_win_cpu():
    with run_agent(config) as [backend, _, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "win_cpu")), "Didn't get win_cpu datapoints"
        for metric in metrics:
            assert wait_for(p(has_datapoint_with_metric_name, backend, metric)), "Didn't get metric %s" % metric

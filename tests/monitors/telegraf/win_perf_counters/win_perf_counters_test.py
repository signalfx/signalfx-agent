from functools import partial as p
import os
import pytest
import string
import sys

from tests.helpers.util import wait_for, run_agent, run_service, container_ip
from tests.helpers.assertions import *

pytestmark = [pytest.mark.windows, pytest.mark.telegraf, pytest.mark.win_perf_counters]

config = string.Template("""
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
""")


@pytest.mark.skipif(sys.platform != 'win32', reason="only runs on windows")
def test_win_perf_counters():
    with run_agent(config) as [backend, _, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "win_perf_counters")), "Didn't get win_perf_counters datapoints"

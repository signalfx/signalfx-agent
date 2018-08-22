from functools import partial as p
from textwrap import dedent
import pytest
import sys

from tests.helpers.util import wait_for, run_agent
from tests.helpers.assertions import *

pytestmark = [
    pytest.mark.skipif(sys.platform != 'win32', reason="only runs on windows"),
    pytest.mark.windows,
    pytest.mark.telegraf,
    pytest.mark.win_perf_counters
]


def test_win_cpu():
    config = dedent("""
        monitors:
         - type: telegraf/win_perf_counters
           printValid: true
           useWildCardExpansion: true
           objects:
            - objectName: "Processor"
              instances:
               - "*"
              counters:
               - "*"
              includeTotal: true
              measurement: "win_cpu"
        """)
    metrics = [
        "win_cpu.Percent_Idle_Time",
        "win_cpu.Percent_Interrupt_Time",
        "win_cpu.Percent_Privileged_Time",
        "win_cpu.Percent_Processor_Time",
        "win_cpu.Percent_User_Time",
    ]
    with run_agent(config) as [backend, _, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "win_cpu")), "Didn't get win_cpu datapoints"
        assert wait_for(p(has_datapoint_with_dim, backend, "instance", "_Total")), "Didn't get _Total datapoints"
        for metric in metrics:
            assert wait_for(p(has_datapoint_with_metric_name, backend, metric)), "Didn't get metric %s" % metric


def test_win_disk():
    config = dedent("""
        monitors:
         - type: telegraf/win_perf_counters
           printValid: true
           useWildCardExpansion: true
           objects:
            - objectName: "LogicalDisk"
              instances:
               - "*"
              counters:
               - "*"
              includeTotal: true
              measurement: "win_disk"
        """)
    metrics = [
        "win_disk.Percent_Idle_Time",
        "win_disk.Percent_Disk_Time",
        "win_disk.Percent_Disk_Read_Time",
        "win_disk.Percent_Disk_Write_Time",
        "win_disk.Current_Disk_Queue_Length",
    ]
    with run_agent(config) as [backend, _, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "win_disk")), "Didn't get win_disk datapoints"
        assert wait_for(p(has_datapoint_with_dim, backend, "instance", "_Total")), "Didn't get _Total datapoints"
        for metric in metrics:
            assert wait_for(p(has_datapoint_with_metric_name, backend, metric)), "Didn't get metric %s" % metric

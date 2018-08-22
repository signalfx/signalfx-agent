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
    with run_agent(config) as [backend, get_output, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "win_cpu")), "Didn't get win_cpu datapoints"
        assert wait_for(p(has_datapoint_with_dim, backend, "instance", "_Total")), "Didn't get _Total datapoints"
        for metric in metrics:
            assert wait_for(p(has_datapoint_with_metric_name, backend, metric)), "Didn't get metric %s" % metric
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"


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
    with run_agent(config) as [backend, get_output, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "win_disk")), "Didn't get win_disk datapoints"
        assert wait_for(p(has_datapoint_with_dim, backend, "instance", "_Total")), "Didn't get _Total datapoints"
        for metric in metrics:
            assert wait_for(p(has_datapoint_with_metric_name, backend, metric)), "Didn't get metric %s" % metric
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"


def test_win_system():
    config = dedent("""
        monitors:
         - type: telegraf/win_perf_counters
           printValid: true
           useWildCardExpansion: true
           objects:
            - objectName: "System"
              instances:
               - "------"
              counters:
               - "*"
              measurement: "win_system"
        """)
    metrics = [
        "win_system.Context_Switches_persec",
        "win_system.Processes",
        "win_system.System_Calls_persec",
        "win_system.System_Up_Time",
        "win_system.Threads",
    ]
    with run_agent(config) as [backend, get_output, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "win_system")), "Didn't get win_system datapoints"
        for metric in metrics:
            assert wait_for(p(has_datapoint_with_metric_name, backend, metric)), "Didn't get metric %s" % metric
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"


def test_win_mem():
    config = dedent("""
        monitors:
         - type: telegraf/win_perf_counters
           printValid: true
           useWildCardExpansion: true
           objects:
            - objectName: "Memory"
              instances:
               - "------"
              counters:
               - "*"
              measurement: "win_mem"
        """)
    metrics = [
        "win_mem.Available_Bytes",
        "win_mem.Cache_Bytes",
        "win_mem.Committed_Bytes",
        "win_mem.Pages_persec",
        "win_mem.Write_Copies_persec",
    ]
    with run_agent(config) as [backend, get_output, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "win_mem")), "Didn't get win_mem datapoints"
        for metric in metrics:
            assert wait_for(p(has_datapoint_with_metric_name, backend, metric)), "Didn't get metric %s" % metric
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"


def test_win_net():
    config = dedent("""
        monitors:
         - type: telegraf/win_perf_counters
           printValid: true
           useWildCardExpansion: true
           objects:
            - objectName: "Network Interface"
              instances:
               - "*"
              counters:
               - "*"
              measurement: "win_net"
        """)
    metrics = [
        "win_net.Bytes_Received_persec",
        "win_net.Bytes_Sent_persec",
        "win_net.Bytes_Total_persec",
        "win_net.Current_Bandwidth",
        "win_net.Packets_persec",
    ]
    with run_agent(config) as [backend, get_output, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "win_net")), "Didn't get win_net datapoints"
        for metric in metrics:
            assert wait_for(p(has_datapoint_with_metric_name, backend, metric)), "Didn't get metric %s" % metric
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"

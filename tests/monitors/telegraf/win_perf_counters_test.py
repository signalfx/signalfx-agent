from functools import partial as p
import pytest
import string
import sys

from tests.helpers.util import wait_for, run_agent
from tests.helpers.assertions import *

pytestmark = [
    pytest.mark.skipif(sys.platform != 'win32', reason="only runs on windows"),
    pytest.mark.windows,
    pytest.mark.telegraf,
    pytest.mark.win_perf_counters
]

config_template = string.Template("""
monitors:
 - type: telegraf/win_perf_counters
   printValid: true
   useWildCardExpansion: true
   objects:
    - objectName: "$object_name"
      instances:
       - "$instance"
      counters:
       - "*"
      includeTotal: $include_total
      measurement: "$measurement"
""")

params = {
    "win_cpu": {
        "object_name": "Processor",
        "instance": "*",
        "include_total": "true",
        "metrics": ["win_cpu.Percent_Idle_Time",
                    "win_cpu.Percent_Interrupt_Time",
                    "win_cpu.Percent_Privileged_Time",
                    "win_cpu.Percent_Processor_Time",
                    "win_cpu.Percent_User_Time"]},
    "win_disk": {
        "object_name": "LogicalDisk",
        "instance": "*",
        "include_total": "true",
        "metrics": ["win_disk.Percent_Idle_Time",
                    "win_disk.Percent_Disk_Time",
                    "win_disk.Percent_Disk_Read_Time",
                    "win_disk.Percent_Disk_Write_Time",
                    "win_disk.Current_Disk_Queue_Length"]},
    "win_system": {
        "object_name": "System",
        "instance": "------",
        "include_total": "false",
        "metrics": ["win_system.Context_Switches_persec",
                    "win_system.Processes",
                    "win_system.System_Calls_persec",
                    "win_system.System_Up_Time",
                    "win_system.Threads"]},
    "win_mem": {
        "object_name": "Memory",
        "instance": "------",
        "include_total": "false",
        "metrics": ["win_mem.Available_Bytes",
                    "win_mem.Cache_Bytes",
                    "win_mem.Committed_Bytes",
                    "win_mem.Pages_persec",
                    "win_mem.Write_Copies_persec"]},
    "win_net": {
        "object_name": "Network Interface",
        "instance": "*",
        "include_total": "false",
        "metrics": ["win_net.Bytes_Received_persec",
                    "win_net.Bytes_Sent_persec",
                    "win_net.Bytes_Total_persec",
                    "win_net.Current_Bandwidth",
                    "win_net.Packets_persec"]},
}


@pytest.mark.parametrize("measurement", params.keys())
def test_win_perf_counters(measurement):
    config = config_template.substitute(
        measurement=measurement,
        object_name=params[measurement]["object_name"],
        instance=params[measurement]["instance"],
        include_total=params[measurement]["include_total"])
    with run_agent(config) as [backend, get_output, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", measurement)), "Didn't get %s datapoints" % measurement
        if params[measurement]["include_total"] == "true":
            assert wait_for(p(has_datapoint_with_dim, backend, "instance", "_Total")), "Didn't get _Total datapoints"
        for metric in params[measurement]["metrics"]:
            assert wait_for(p(has_datapoint_with_metric_name, backend, metric)), "Didn't get metric %s" % metric
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"

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

metrics = {
    "win_cpu": [
        "win_cpu.Percent_Idle_Time",
        "win_cpu.Percent_Interrupt_Time",
        "win_cpu.Percent_Privileged_Time",
        "win_cpu.Percent_Processor_Time",
        "win_cpu.Percent_User_Time"
    ],
    "win_disk": [
        "win_disk.Percent_Idle_Time",
        "win_disk.Percent_Disk_Time",
        "win_disk.Percent_Disk_Read_Time",
        "win_disk.Percent_Disk_Write_Time",
        "win_disk.Current_Disk_Queue_Length",
    ],
    "win_system": [
        "win_system.Context_Switches_persec",
        "win_system.Processes",
        "win_system.System_Calls_persec",
        "win_system.System_Up_Time",
        "win_system.Threads",
    ],
    "win_mem": [
        "win_mem.Available_Bytes",
        "win_mem.Cache_Bytes",
        "win_mem.Committed_Bytes",
        "win_mem.Pages_persec",
        "win_mem.Write_Copies_persec",
    ],
    "win_net": [
        "win_net.Bytes_Received_persec",
        "win_net.Bytes_Sent_persec",
        "win_net.Bytes_Total_persec",
        "win_net.Current_Bandwidth",
        "win_net.Packets_persec",
    ],
}

params = [
    ("Processor", "*", "true", "win_cpu"),
    ("LogicalDisk", "*", "true", "win_disk"),
    ("System", "------", "false", "win_system"),
    ("Memory", "------", "false", "win_mem"),
    ("Network Interface", "*", "false", "win_net")
]


@pytest.mark.parametrize("object_name, instance, include_total, measurement", params, ids=[p[3] for p in params])
def test_perf_counter(object_name, instance, include_total, measurement):
    config = config_template.substitute(
        object_name=object_name,
        instance=instance,
        include_total=include_total,
        measurement=measurement)
    with run_agent(config) as [backend, get_output, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", measurement)), "Didn't get %s datapoints" % measurement
        if include_total == "true":
            assert wait_for(p(has_datapoint_with_dim, backend, "instance", "_Total")), "Didn't get _Total datapoints"
        for metric in metrics[measurement]:
            assert wait_for(p(has_datapoint_with_metric_name, backend, metric)), "Didn't get metric %s" % metric
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"

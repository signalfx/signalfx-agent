from functools import partial as p
import pytest
import string

from helpers.util import wait_for, run_agent
from helpers.assertions import *

pytestmark = [pytest.mark.windows_only, pytest.mark.windows, pytest.mark.telegraf, pytest.mark.win_perf_counters]

config_template = string.Template(
    """
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
"""
)


def get_config(measurement, object_name, instance, include_total):
    return config_template.substitute(
        measurement=measurement, object_name=object_name, instance=instance, include_total=str(include_total).lower()
    )


@pytest.fixture
def win_cpu():
    measurement = "win_cpu"
    include_total = True
    config = get_config(measurement, "Processor", "*", include_total)
    metrics = [
        "win_cpu.Percent_Idle_Time",
        "win_cpu.Percent_Interrupt_Time",
        "win_cpu.Percent_Privileged_Time",
        "win_cpu.Percent_Processor_Time",
        "win_cpu.Percent_User_Time",
    ]
    return measurement, config, include_total, metrics


@pytest.fixture
def win_disk():
    measurement = "win_disk"
    include_total = True
    config = get_config(measurement, "LogicalDisk", "*", include_total)
    metrics = [
        "win_disk.Percent_Idle_Time",
        "win_disk.Percent_Disk_Time",
        "win_disk.Percent_Disk_Read_Time",
        "win_disk.Percent_Disk_Write_Time",
        "win_disk.Current_Disk_Queue_Length",
    ]
    return measurement, config, include_total, metrics


@pytest.fixture
def win_mem():
    measurement = "win_mem"
    include_total = False
    config = get_config(measurement, "Memory", "------", include_total)
    metrics = [
        "win_mem.Available_Bytes",
        "win_mem.Cache_Bytes",
        "win_mem.Committed_Bytes",
        "win_mem.Pages_persec",
        "win_mem.Write_Copies_persec",
    ]
    return measurement, config, include_total, metrics


@pytest.fixture
def win_net():
    measurement = "win_net"
    include_total = False
    config = get_config(measurement, "Network Interface", "*", include_total)
    metrics = [
        "win_net.Bytes_Received_persec",
        "win_net.Bytes_Sent_persec",
        "win_net.Bytes_Total_persec",
        "win_net.Current_Bandwidth",
        "win_net.Packets_persec",
    ]
    return measurement, config, include_total, metrics


@pytest.fixture
def win_system():
    measurement = "win_system"
    include_total = False
    config = get_config(measurement, "System", "------", include_total)
    metrics = [
        "win_system.Context_Switches_persec",
        "win_system.Processes",
        "win_system.System_Calls_persec",
        "win_system.System_Up_Time",
        "win_system.Threads",
    ]
    return measurement, config, include_total, metrics


@pytest.fixture(params=["win_cpu", "win_disk", "win_mem", "win_net", "win_system"])
def monitor_config(request):
    return request.getfixturevalue(request.param)


def test_win_perf_counters(monitor_config):
    measurement, config, include_total, metrics = monitor_config
    with run_agent(config) as [backend, get_output, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", measurement)), (
            "Didn't get %s datapoints" % measurement
        )
        if include_total:
            assert wait_for(p(has_datapoint_with_dim, backend, "instance", "_Total")), "Didn't get _Total datapoints"
        for metric in metrics:
            assert wait_for(p(has_datapoint_with_metric_name, backend, metric)), "Didn't get metric %s" % metric
        assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"

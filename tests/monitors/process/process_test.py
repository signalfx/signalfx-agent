import os

import psutil
import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint
from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify

pytestmark = [pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("process")


def test_process_monitor_process_name_filter():
    proc = psutil.Process(os.getpid())

    self_proc_name = proc.name()

    with Agent.run(
        f"""
monitors:
  - type: process
    processes:
     - {self_proc_name}
"""
    ) as agent:
        verify(agent, METADATA.all_metrics)
        assert has_datapoint(agent.fake_services, dimensions={"command": self_proc_name})


def test_process_monitor_executable_filter():
    proc = psutil.Process(os.getpid())

    self_proc_exec = proc.exe()

    with Agent.run(
        f"""
monitors:
  - type: process
    executables:
     - {self_proc_exec}
"""
    ) as agent:
        verify(agent, METADATA.all_metrics)
        assert has_datapoint(agent.fake_services, dimensions={"executable": self_proc_exec})

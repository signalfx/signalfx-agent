from functools import partial as p
import pytest

from helpers.util import *
from helpers.assertions import *


pytestmark = [pytest.mark.windows, pytest.mark.telegraf_procstat, pytest.mark.telegraf]

monitor_config = """
monitors:
  - type: telegraf/procstat
    exe: "signalfx-agent*"
"""


def test_telegraf_procstat():
    with run_agent(monitor_config) as [backend, _, _]:
        # wait for fake ingest to receive the procstat metrics
        assert wait_for(
            p(has_datapoint_with_metric_name, backend, "procstat.cpu_usage")
        ), "no cpu usage datapoint found for process"
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "procstat")), "plugin dimension not set"

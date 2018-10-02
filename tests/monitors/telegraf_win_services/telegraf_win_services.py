from functools import partial as p
import pytest
import string
import tempfile

from tests.helpers.util import wait_for, run_agent
from tests.helpers.assertions import has_datapoint_with_dim


pytestmark = [pytest.mark.windows,
              pytest.mark.win_services,
              pytest.mark.telegraf]

monitor_config = string.Template("""
monitors:
  - type: telegraf/win_services
""")


def test_win_services():
        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "win_services")), "didn't get datapoint with expected plugin dimension"


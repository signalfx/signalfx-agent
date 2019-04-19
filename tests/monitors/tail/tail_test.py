import string
import tempfile
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.util import wait_for

pytestmark = [pytest.mark.windows, pytest.mark.tail, pytest.mark.telegraf]

MONITOR_CONFIG = string.Template(
    """
monitors:
  - type: telegraf/tail
    files:
     - '$file'
    watchMethod: poll       # specify the file watch method ("inotify" or "poll")
    fromBeginning: true     # specify to read from the beginning
    telegrafParser:         # configure the telegrafParser
      dataFormat: influx  # set the parseer format to "influx"
"""
)


def test_tail():
    with tempfile.NamedTemporaryFile("w+b") as tmpfile:
        config = MONITOR_CONFIG.substitute(file=tmpfile.name)
        tmpfile.write(b"disk,customtag1=foo bytes=1024\n")
        tmpfile.flush()
        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "customtag1", "foo")
            ), "didn't get datapoint written before startup"
            tmpfile.write(b"mem,customtag2=foo2 bytes=1024\n")
            tmpfile.flush()
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "customtag2", "foo2")
            ), "didn't get datapoint written after startup"
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "telegraf-tail")
            ), "didn't get datapoint with expected plugin dimension"

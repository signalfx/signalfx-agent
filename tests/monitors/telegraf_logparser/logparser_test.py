import string
import tempfile
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.util import wait_for

pytestmark = [pytest.mark.windows, pytest.mark.logparser, pytest.mark.telegraf]

MONITOR_CONFIG = string.Template(
    """
monitors:
  - type: telegraf/logparser
    files:
     - '$file'
    watchMethod: poll       # specify the file watch method ("inotify" or "poll")
    fromBeginning: true     # specify to read from the beginning
    measurementName: test-measurement
    patterns:
     - "%{COMMON_LOG_FORMAT}"
    timezone: UTC
"""
)


def test_logparser():
    with tempfile.NamedTemporaryFile("w+b") as tmpfile:
        config = MONITOR_CONFIG.substitute(file=tmpfile.name)
        tmpfile.write(
            b'127.0.0.1 - charlie [02/Oct/2018:13:00:30 -0700] "GET /apache_stuff.gif HTTP/1.0" 200 4321 '
            b'"http://www.signalfx.com" "Mozilla/4.08 (Macintosh; I; PPC)" \n'
        )
        tmpfile.flush()
        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "resp_code", "200")
            ), "didn't get datapoint written before startup"
            tmpfile.write(
                b'127.0.0.1 - charlie [02/Oct/2018:13:00:30 -0700] "GET /apache_stuff.gif HTTP/1.0" 404 4321 '
                b'"http://www.signalfx.com" "Mozilla/4.08 (Macintosh; I; PPC)" \n'
            )
            tmpfile.flush()
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "resp_code", "404")
            ), "didn't get datapoint written after startup"
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "telegraf-logparser")
            ), "didn't get datapoint with expected plugin dimension"

from functools import partial as p
import pytest
import string
import tempfile

from tests.helpers.util import wait_for, run_agent
from tests.helpers.assertions import has_datapoint_with_dim


pytestmark = [pytest.mark.windows,
              pytest.mark.logparser,
              pytest.mark.telegraf]

monitor_config = string.Template("""
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
""")


def test_logparser():
    with tempfile.NamedTemporaryFile('w+b') as f:
        config = monitor_config.substitute(file=f.name)
        f.write(b'127.0.0.1 - charlie [02/Oct/2018:13:00:30 -0700] "GET /apache_stuff.gif HTTP/1.0" 200 4321 "http://www.signalfx.com" "Mozilla/4.08 (Macintosh; I; PPC)" \n')
        f.flush()
        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "resp_code", "200")), "didn't get datapoint written before startup"
            f.write(b'127.0.0.1 - charlie [02/Oct/2018:13:00:30 -0700] "GET /apache_stuff.gif HTTP/1.0" 404 4321 "http://www.signalfx.com" "Mozilla/4.08 (Macintosh; I; PPC)" \n')
            f.flush()
            assert wait_for(p(has_datapoint_with_dim, backend, "resp_code", "404")), "didn't get datapoint written after startup"
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "telegraf-logparser")), "didn't get datapoint with expected plugin dimension"


from functools import partial as p
import pytest
import string
import tempfile

from helpers.util import wait_for, run_agent
from helpers.assertions import has_datapoint_with_dim


pytestmark = [pytest.mark.windows,
              pytest.mark.tail,
              pytest.mark.telegraf]

monitor_config = string.Template("""
monitors:
  - type: telegraf/tail
    files:
     - '$file'
    watchMethod: poll       # specify the file watch method ("inotify" or "poll")
    fromBeginning: true     # specify to read from the beginning
    telegrafParser:         # configure the telegrafParser
      dataFormat: influx  # set the parseer format to "influx"
""")


def test_tail():
    with tempfile.NamedTemporaryFile('w+b') as f:
        config = monitor_config.substitute(file=f.name)
        f.write(b'disk,customtag1=foo bytes=1024\n')
        f.flush()
        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "customtag1", "foo")), "didn't get datapoint written before startup"
            f.write(b'mem,customtag2=foo2 bytes=1024\n')
            f.flush()
            assert wait_for(p(has_datapoint_with_dim, backend, "customtag2", "foo2")), "didn't get datapoint written after startup"
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "telegraf-tail")), "didn't get datapoint with expected plugin dimension"


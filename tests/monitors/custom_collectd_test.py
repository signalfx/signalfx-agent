from functools import partial as p
import os
import string

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, run_agent
from tests.helpers.assertions import *

config = """
monitors:
  - type: collectd/custom
    template: |
      LoadPlugin "ping"
      <Plugin ping>
        Host "google.com"
      </Plugin>

"""

def test_custom_collectd():
    with run_agent(config) as [backend, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "ping")), "Didn't get ping datapoints"

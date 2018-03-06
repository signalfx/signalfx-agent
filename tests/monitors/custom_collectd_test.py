from functools import partial as p
import os
import string

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, run_agent
from tests.helpers.assertions import *

def test_custom_collectd():
    with run_agent("""
monitors:
  - type: collectd/df
  - type: collectd/custom
    template: |
      LoadPlugin "ping"
      <Plugin ping>
        Host "google.com"
      </Plugin>
""") as [backend, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "ping")), "Didn't get ping datapoints"
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "df")), "Didn't get df datapoints"


def test_custom_collectd_multiple_templates():
    with run_agent("""
monitors:
  - type: collectd/df
  - type: collectd/custom
    templates:
     - |
       LoadPlugin "cpu"
     - |
       LoadPlugin "ping"
       <Plugin ping>
         Host "google.com"
       </Plugin>
collectd:
  logLevel: debug
""") as [backend, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "df")), "Didn't get df datapoints"
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "cpu")), "Didn't get cpufreq datapoints"
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "ping")), "Didn't get ping datapoints"

import os

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, run_agent

basic_config = """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/cpu
  - type: collectd/uptime
"""

def test_basic():
    with run_agent(basic_config) as [backend, _]:
        assert wait_for(lambda: len(backend.datapoints) > 0), "Didn't get any datapoints"





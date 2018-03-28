from functools import partial as p
import os

from tests.helpers import fake_backend
from tests.helpers.util import ensure_always, wait_for, run_agent
from tests.helpers.assertions import *


def test_sets_hostname():
    with run_agent("""
hostname: acmeinc.com
monitors:
  - type: collectd/signalfx-metadata
    persistencePath: /dev/null
  - type: collectd/cpu
  - type: collectd/uptime
    """) as [backend, get_output, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "host", "acmeinc.com")), "Didn't get overridden hostname in datapoint"
        assert wait_for(p(has_event_with_dim, backend, "host", "acmeinc.com"), 30), "Didn't get overridden hostname in event"


def test_does_not_set_hostname_if_not_host_specific():
    with run_agent("""
hostname: acmeinc.com
disableHostDimensions: true
monitors:
  - type: collectd/signalfx-metadata
    persistencePath: /dev/null
  - type: collectd/cpu
  - type: collectd/uptime
    """) as [backend, get_output, _]:
        assert ensure_always(lambda: not has_datapoint_with_dim(backend, "host", "acmeinc.com")), "Got overridden hostname in datapoint"
        assert ensure_always(lambda: not has_event_with_dim(backend, "host", "acmeinc.com")), "Got overridden hostname in event"

def test_does_not_set_hostname_on_monitor_if_not_host_specific():
    with run_agent("""
hostname: acmeinc.com
monitors:
  - type: collectd/signalfx-metadata
    persistencePath: /dev/null
  - type: collectd/cpu
  - type: collectd/uptime
    disableHostDimensions: true
    """) as [backend, get_output, _]:
        assert wait_for(p(has_datapoint_with_all_dims, backend, dict(host="acmeinc.com", plugin="signalfx-metadata"))), "Didn't get overridden hostname in datapoint"

        assert ensure_always(lambda: not has_datapoint_with_dim(backend, "uptime", "acmeinc.com")), "Got overridden hostname in datapoint"

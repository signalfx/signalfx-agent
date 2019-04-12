from functools import partial as p

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_all_dims, has_datapoint_with_dim, has_event_with_dim
from tests.helpers.util import ensure_always, wait_for


def test_sets_hostname():
    with Agent.run(
        """
hostname: acmeinc.com
monitors:
  - type: collectd/signalfx-metadata
    persistencePath: /dev/null
  - type: collectd/cpu
  - type: collectd/uptime
    """
    ) as agent:
        assert wait_for(
            p(has_datapoint_with_dim, agent.fake_services, "host", "acmeinc.com")
        ), "Didn't get overridden hostname in datapoint"
        assert wait_for(
            p(has_event_with_dim, agent.fake_services, "host", "acmeinc.com"), 30
        ), "Didn't get overridden hostname in event"


def test_does_not_set_hostname_if_not_host_specific():
    with Agent.run(
        """
hostname: acmeinc.com
disableHostDimensions: true
monitors:
  - type: collectd/signalfx-metadata
    persistencePath: /dev/null
  - type: collectd/cpu
  - type: collectd/uptime
    """
    ) as agent:
        assert ensure_always(
            lambda: not has_datapoint_with_dim(agent.fake_services, "host", "acmeinc.com")
        ), "Got overridden hostname in datapoint"
        assert ensure_always(
            lambda: not has_event_with_dim(agent.fake_services, "host", "acmeinc.com")
        ), "Got overridden hostname in event"


def test_does_not_set_hostname_on_monitor_if_not_host_specific():
    with Agent.run(
        """
hostname: acmeinc.com
monitors:
  - type: collectd/signalfx-metadata
    persistencePath: /dev/null
  - type: collectd/cpu
  - type: collectd/uptime
    disableHostDimensions: true
    """
    ) as agent:
        assert wait_for(
            p(has_datapoint_with_all_dims, agent.fake_services, dict(host="acmeinc.com", plugin="signalfx-metadata"))
        ), "Didn't get overridden hostname in datapoint"

        assert ensure_always(
            lambda: not has_datapoint_with_dim(agent.fake_services, "uptime", "acmeinc.com")
        ), "Got overridden hostname in datapoint"

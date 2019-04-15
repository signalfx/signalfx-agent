import time
from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.util import ensure_always, wait_for

pytestmark = [
    pytest.mark.collectd,
    pytest.mark.custom,
    pytest.mark.custom_collectd,
    pytest.mark.monitor_without_endpoints,
]


def test_custom_collectd():
    with Agent.run(
        """
monitors:
  - type: collectd/df
  - type: collectd/custom
    template: |
      LoadPlugin "ping"
      <Plugin ping>
        Host "google.com"
      </Plugin>
"""
    ) as agent:
        assert wait_for(p(has_datapoint_with_dim, agent.fake_services, "plugin", "ping")), "Didn't get ping datapoints"
        assert wait_for(p(has_datapoint_with_dim, agent.fake_services, "plugin", "df")), "Didn't get df datapoints"


def test_custom_collectd_multiple_templates():
    with Agent.run(
        """
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
"""
    ) as agent:
        assert wait_for(p(has_datapoint_with_dim, agent.fake_services, "plugin", "df")), "Didn't get df datapoints"
        assert wait_for(p(has_datapoint_with_dim, agent.fake_services, "plugin", "cpu")), "Didn't get cpu datapoints"
        assert wait_for(p(has_datapoint_with_dim, agent.fake_services, "plugin", "ping")), "Didn't get ping datapoints"


def test_custom_collectd_shutdown():
    with Agent.run(
        dedent(
            """
        monitors:
          - type: collectd/df
          - type: collectd/custom
            template: |
              LoadPlugin "ping"
              <Plugin ping>
                Host "google.com"
              </Plugin>
    """
        )
    ) as agent:
        assert wait_for(p(has_datapoint_with_dim, agent.fake_services, "plugin", "ping")), "Didn't get ping datapoints"
        assert wait_for(p(has_datapoint_with_dim, agent.fake_services, "plugin", "df")), "Didn't get df datapoints"

        agent.update_config(
            dedent(
                """
            monitors:
              - type: collectd/df
        """
            )
        )

        time.sleep(3)
        agent.fake_services.reset_datapoints()

        assert ensure_always(
            lambda: not has_datapoint_with_dim(agent.fake_services, "plugin", "ping")
        ), "Got ping datapoint when we shouldn't have"

        agent.update_config(
            dedent(
                """
            monitors:
              - type: collectd/df
              - type: collectd/custom
                template: |
                  LoadPlugin "ping"
                  <Plugin ping>
                    Host "google.com"
                  </Plugin>
        """
            )
        )

        assert wait_for(p(has_datapoint_with_dim, agent.fake_services, "plugin", "ping")), "Didn't get ping datapoints"

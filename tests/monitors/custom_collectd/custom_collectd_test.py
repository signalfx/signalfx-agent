import time
from functools import partial as p
from textwrap import dedent

import pytest

from helpers.assertions import has_datapoint_with_dim
from helpers.util import ensure_always, run_agent, wait_for

pytestmark = [
    pytest.mark.collectd,
    pytest.mark.custom,
    pytest.mark.custom_collectd,
    pytest.mark.monitor_without_endpoints,
]


def test_custom_collectd():
    with run_agent(
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
    ) as [backend, _, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "ping")), "Didn't get ping datapoints"
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "df")), "Didn't get df datapoints"


def test_custom_collectd_multiple_templates():
    with run_agent(
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
    ) as [backend, _, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "df")), "Didn't get df datapoints"
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "cpu")), "Didn't get cpu datapoints"
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "ping")), "Didn't get ping datapoints"


def test_custom_collectd_shutdown():
    with run_agent(
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
    ) as [backend, _, configure]:
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "ping")), "Didn't get ping datapoints"
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "df")), "Didn't get df datapoints"

        configure(
            dedent(
                """
            monitors:
              - type: collectd/df
        """
            )
        )

        time.sleep(3)
        backend.datapoints.clear()

        assert ensure_always(
            lambda: not has_datapoint_with_dim(backend, "plugin", "ping")
        ), "Got ping datapoint when we shouldn't have"

        configure(
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

        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "ping")), "Didn't get ping datapoints"

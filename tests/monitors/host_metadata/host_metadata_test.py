from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_metric_name, has_event_with_dim
from tests.helpers.util import wait_for

pytestmark = [pytest.mark.host_metadata, pytest.mark.monitor_without_endpoints]

MONITOR_CONFIG = """
monitors:
  - type: host-metadata
"""


def test_host_metadata_monitor():
    with Agent.run(MONITOR_CONFIG) as agent:
        assert wait_for(
            p(has_datapoint_with_metric_name, agent.fake_services, "sfxagent.hostmetadata")
        ), "Didn't get agent hostmetadata datapoints"
        # wait for up to 90 seconds to receive metadata properties
        # they are guaranteed to emit in with in the first minute
        assert wait_for(
            p(has_event_with_dim, agent.fake_services, "plugin", "signalfx-metadata"), 90
        ), "Didn't get metadata properties"

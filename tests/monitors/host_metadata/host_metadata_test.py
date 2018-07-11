from functools import partial as p
import pytest

from tests.helpers.util import wait_for, run_agent
from tests.helpers.assertions import has_datapoint_with_metric_name, has_event_with_dim

pytestmark = [pytest.mark.host_metadata, pytest.mark.monitor_without_endpoints]

monitor_config = """
monitors:
  - type: host-metadata
"""


def test_host_metadata_monitor():
    with run_agent(monitor_config) as [backend, _, _]:
        assert wait_for(p(has_datapoint_with_metric_name, backend,
                        "sfxagent.hostmetadata")), "Didn't get agent hostmetadata datapoints"
        # wait for up to 90 seconds to receive metadata properties
        # they are guaranteed to emit in with in the first minute
        assert wait_for(p(has_event_with_dim, backend, "plugin",
                        "signalfx-metadata"), 90), "Didn't get metadata properties"

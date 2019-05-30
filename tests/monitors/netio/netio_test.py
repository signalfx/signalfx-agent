from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import any_metric_found, has_datapoint, has_log_message
from tests.helpers.metadata import Metadata
from tests.helpers.util import ensure_never, wait_for
from tests.helpers.verify import verify

pytestmark = [pytest.mark.windows, pytest.mark.netio, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("netio")


def test_netio_defaults():
    with Agent.run(
        """
    monitors:
      - type: net-io
    """
    ) as agent:
        verify(agent, METADATA.default_metrics)
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_netio_filter():
    forbidden_metrics = METADATA.default_metrics - {"network.total"}

    with Agent.run(
        """
    procPath: /proc
    monitors:
      - type: net-io
        interfaces:
         - "!*"
    """
    ) as agent:
        assert wait_for(
            p(has_datapoint, agent.fake_services, metric_name="network.total"), timeout_seconds=60
        ), "timed out waiting for metrics and/or dimensions!"
        assert ensure_never(p(any_metric_found, agent.fake_services, forbidden_metrics))
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

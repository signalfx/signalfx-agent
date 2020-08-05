from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.metadata import Metadata
from tests.helpers.util import wait_for
from tests.helpers.verify import run_agent_verify_default_metrics

pytestmark = [pytest.mark.windows, pytest.mark.filesystems, pytest.mark.monitor_without_endpoints]

MONITOR = "ntp"
METADATA = Metadata.from_package(MONITOR)
HOST = "pool.ntp.org"


@pytest.mark.flaky(reruns=2, reruns_delay=30)
def test_default_metrics():
    # Config to get every possible metrics
    agent_config = dedent(
        f"""
        monitors:
        - type: ntp
          host: {HOST}
        """
    )
    # every metrics should be reported for https site
    run_agent_verify_default_metrics(agent_config, METADATA)


@pytest.mark.flaky(reruns=2, reruns_delay=30)
def test_min_interval():
    # Config to get every possible dimensions (and metrics so) to OK
    with Agent.run(
        f"""
        monitors:
        - type: ntp
          host: {HOST}
        """
    ) as agent:
        # configured host should be in dimension of metric
        assert wait_for(
            p(has_datapoint_with_dim, agent.fake_services, MONITOR, HOST)
        ), "Didn't get ntp datapoints with {}:{} dimension".format(MONITOR, HOST)
        # should have only one metric while default interval should be enforced
        if len(METADATA.default_metrics) != len(agent.fake_services.datapoints):
            assert False

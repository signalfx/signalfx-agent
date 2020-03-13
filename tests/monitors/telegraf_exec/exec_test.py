from functools import partial as p
from pathlib import Path

import pytest
from signalfx.generated_protocol_buffers import signal_fx_protocol_buffers_pb2 as sf_pbuf
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import wait_for

pytestmark = [pytest.mark.telegraf]

TEST_DIR = Path(__file__).parent


def test_telegraf_exec_basic():
    with Agent.run(
        f"""
    monitors:
     - type: telegraf/exec
       command: {TEST_DIR / "script.sh"}
       signalFxCumulativeMetrics:
        - weather.lightning_strikes
       telegrafParser:
         dataFormat: influx
    """
    ) as agent:
        assert wait_for(
            p(
                has_datapoint,
                agent.fake_services,
                "weather.temperature",
                metric_type=sf_pbuf.GAUGE,
                dimensions={"location": "us-midwest"},
                value=70,
            )
        )
        assert wait_for(
            p(has_datapoint, agent.fake_services, "weather.temperature", dimensions={"location": "us-east"}, value=80)
        )
        assert wait_for(
            p(has_datapoint, agent.fake_services, "weather.temperature", dimensions={"location": "us-west"}, value=75)
        )
        assert wait_for(
            p(has_datapoint, agent.fake_services, "weather.temperature", dimensions={"location": "us-west"}, value=75)
        )
        assert wait_for(
            p(
                has_datapoint,
                agent.fake_services,
                "weather.thunderclaps",
                dimensions={"location": "us-west"},
                metric_type=sf_pbuf.CUMULATIVE_COUNTER,
                value=5,
            )
        )
        assert wait_for(
            p(
                has_datapoint,
                agent.fake_services,
                "weather.lightning_strikes",
                dimensions={"location": "us-east"},
                metric_type=sf_pbuf.CUMULATIVE_COUNTER,
                value=8,
            )
        )

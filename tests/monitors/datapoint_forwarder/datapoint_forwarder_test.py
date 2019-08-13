"""
Tests the tracing forwarder monitor
"""
import json
import random
from functools import partial as p
from textwrap import dedent

import pytest
import requests
from signalfx.generated_protocol_buffers import signal_fx_protocol_buffers_pb2 as sf_pbuf
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, tcp_port_open_locally
from tests.helpers.util import wait_for

pytestmark = [pytest.mark.trace_forwarder, pytest.mark.monitor_without_endpoints]

TEST_DATAPOINTS = {
    "gauge": [
        {"metric": "my_metric", "dimensions": {"env": "prod"}, "timestamp": 5_000_000, "value": 4000},
        {"metric": "other_metric", "dimensions": {"env": "dev"}, "timestamp": 5_000_000, "value": 5000},
    ]
}


# @pytest.mark.flaky(reruns=2)  # Flaky due to the random port assignment that might conflict
def test_forwarder_dataponts_json():
    """
    Test that the forwarder can consume json datapoints
    """
    port = random.randint(5001, 20000)
    with Agent.run(
        dedent(
            f"""
        hostname: "testhost"
        monitors:
          - type: signalfx-forwarder
            listenAddress: localhost:{port}
    """
        )
    ) as agent:
        assert wait_for(p(tcp_port_open_locally, port)), "datapoint forwarder port never opened!"
        resp = requests.post(
            f"http://localhost:{port}/v2/datapoint",
            headers={"Content-Type": "application/json"},
            data=json.dumps(TEST_DATAPOINTS),
        )

        assert resp.status_code == 200, f"Bad response: {resp.content}"

        assert wait_for(
            p(has_datapoint, agent.fake_services, dimensions={"env": "prod"})
        ), "Didn't get datapoint with env=prod"
        assert wait_for(
            p(has_datapoint, agent.fake_services, dimensions={"env": "dev"})
        ), "Didn't get datapoint with env=dev"


# pylint: disable=no-member
# @pytest.mark.flaky(reruns=2)  # Flaky due to the random port assignment that might conflict
def test_forwarder_datapoints_protobuf():
    """
    Test that the forwarder can consume protobuf datapoints
    """
    port = random.randint(5001, 20000)
    with Agent.run(
        dedent(
            f"""
        hostname: "testhost"
        monitors:
          - type: signalfx-forwarder
            listenAddress: localhost:{port}
    """
        )
    ) as agent:
        assert wait_for(p(tcp_port_open_locally, port)), "datapoint forwarder port never opened!"

        dpum = sf_pbuf.DataPointUploadMessage()
        for typ, dps in TEST_DATAPOINTS.items():
            for dp in dps:
                pbuf_dp = sf_pbuf.DataPoint()
                pbuf_dp.metricType = getattr(sf_pbuf, typ.upper())
                pbuf_dp.value.intValue = dp["value"]
                pbuf_dp.metric = dp["metric"]
                for key, value in dp.get("dimensions", {}).items():
                    dim = pbuf_dp.dimensions.add()
                    dim.key = key
                    dim.value = value
                dpum.datapoints.extend([pbuf_dp])

        resp = requests.post(
            f"http://localhost:{port}/v2/datapoint",
            headers={"Content-Type": "application/x-protobuf"},
            data=dpum.SerializeToString(),
        )

        assert resp.status_code == 200, f"Bad response: {resp.content}"

        assert wait_for(
            p(has_datapoint, agent.fake_services, dimensions={"env": "prod"})
        ), "Didn't get datapoint with env=prod"
        assert wait_for(
            p(has_datapoint, agent.fake_services, dimensions={"env": "dev"})
        ), "Didn't get datapoint with env=dev"

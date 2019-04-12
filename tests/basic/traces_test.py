"""
Tests the trace span correlation logic in the writer
"""
import json
import random
import time
from functools import partial as p
from textwrap import dedent

import requests
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, has_trace_span, tcp_port_open_locally
from tests.helpers.util import REPO_ROOT_DIR, ensure_never, wait_for


# Make this a function so it returns a fresh copy on each call
def _test_trace():
    return [
        {
            "traceId": "0123456789abcdef",
            "name": "get",
            "id": "abcdef0123456789",
            "kind": "CLIENT",
            "timestamp": 1_538_406_065_536_000,
            "duration": 10000,
            "localEndpoint": {"serviceName": "myapp", "ipv4": "10.0.0.1"},
            "tags": {"env": "prod"},
        },
        {
            "traceId": "0123456789abcdef",
            "name": "fetch",
            "parentId": "abcdef0123456789",
            "id": "def0123456789abc",
            "kind": "SERVER",
            "timestamp": 1_538_406_068_536_000,
            "duration": 5000,
            "localEndpoint": {"serviceName": "file-server", "ipv4": "10.0.0.2"},
            "tags": {"env": "prod", "file": "test.pdf"},
        },
    ]


def test_tracing_output():
    """
    Test that the basic trace writer and service tracker work
    """
    port = random.randint(5001, 20000)
    with Agent.run(
        dedent(
            f"""
        hostname: "testhost"
        writer:
            traceHostCorrelationMetricsInterval: 1s
            staleServiceTimeout: 5s
        monitors:
          - type: trace-forwarder
            listenAddress: localhost:{port}
        metricsToExclude:
          - {{"#from": "{REPO_ROOT_DIR}/whitelist.json", flatten: true}}
    """
        )
    ) as agent:
        assert wait_for(p(tcp_port_open_locally, port)), "trace forwarder port never opened!"
        resp = requests.post(
            f"http://localhost:{port}/v1/trace",
            headers={"Content-Type": "application/json"},
            data=json.dumps(_test_trace()),
        )

        assert resp.status_code == 200

        assert wait_for(p(has_trace_span, agent.fake_services, tags={"env": "prod"})), "Didn't get span tag"

        assert wait_for(p(has_trace_span, agent.fake_services, name="fetch")), "Didn't get span name"

        assert wait_for(
            p(
                has_datapoint,
                agent.fake_services,
                metric_name="sf.int.service.heartbeat",
                dimensions={"sf_hasService": "myapp", "host": "testhost"},
            )
        ), "Didn't get host correlation datapoint"

        # Service names expire after 5s in the config provided in this test
        time.sleep(8)
        agent.fake_services.reset_datapoints()

        assert ensure_never(
            p(has_datapoint, agent.fake_services, metric_name="sf.int.service.heartbeat"), timeout_seconds=5
        ), "Got infra correlation metric when it should have been expired"


def test_tracing_load():
    """
    Test that all of the traces sent through the agent get the proper service
    correlation datapoint.
    """
    port = random.randint(5001, 20000)
    with Agent.run(
        dedent(
            f"""
        hostname: "testhost"
        writer:
            sendTraceHostCorrelationMetrics: true
            traceHostCorrelationMetricsInterval: 1s
            staleServiceTimeout: 7s
        monitors:
          - type: trace-forwarder
            listenAddress: localhost:{port}
    """
        )
    ) as agent:
        assert wait_for(p(tcp_port_open_locally, port)), "trace forwarder port never opened!"
        for i in range(0, 100):
            spans = _test_trace()
            spans[0]["localEndpoint"]["serviceName"] += f"-{i}"
            spans[1]["localEndpoint"]["serviceName"] += f"-{i}"
            resp = requests.post(
                f"http://localhost:{port}/v1/trace",
                headers={"Content-Type": "application/json"},
                data=json.dumps(spans),
            )

            assert resp.status_code == 200

        for i in range(0, 100):
            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="sf.int.service.heartbeat",
                    dimensions={"sf_hasService": f"myapp-{i}", "host": "testhost"},
                )
            ), "Didn't get host correlation datapoint"

            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="sf.int.service.heartbeat",
                    dimensions={"sf_hasService": f"file-server-{i}", "host": "testhost"},
                )
            ), "Didn't get host correlation datapoint"

        time.sleep(10)
        agent.fake_services.reset_datapoints()

        assert ensure_never(
            p(has_datapoint, agent.fake_services, metric_name="sf.int.service.heartbeat"), timeout_seconds=5
        ), "Got infra correlation metric when it should have been expired"

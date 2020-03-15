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
from tests.helpers.assertions import has_datapoint, has_trace_span, tcp_port_open_locally, has_dim_set_prop, not_has_dim_set_prop
from tests.helpers.util import ensure_never, retry_on_ebadf, wait_for


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
            "tags": {"env": "prod", "environment": "prod", "file": "test.pdf"},
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

        assert wait_for(
            p(
                has_dim_set_prop,
                agent.fake_services,
                dim_name="host",
                dim_value="testhost",
                prop_name="sf_services",
                prop_values=["myapp", "file-server"]
            ), "Didn't get infrastructure correlation property"
        )

        # Service names expire after 5s in the config provided in this test
        time.sleep(8)
        agent.fake_services.reset_datapoints()

        assert ensure_never(
            p(has_datapoint, agent.fake_services, metric_name="sf.int.service.heartbeat"), timeout_seconds=5
        ), "Got infra correlation metric when it should have been expired"

        assert wait_for(
            p(
                not_has_dim_set_prop,
                agent.fake_services,
                dim_name="host",
                dim_value="testhost",
                prop_name="sf_services",
                prop_values=["myapp", "file-server"]
            )
        ), "Got correlation property when it should have been expired"


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
            resp = retry_on_ebadf(
                lambda: requests.post(
                    f"http://localhost:{port}/v1/trace",
                    headers={"Content-Type": "application/json"},
                    data=json.dumps(spans),  # pylint:disable=cell-var-from-loop
                )
            )()

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

            assert wait_for(
                p(
                    has_dim_set_prop,
                    agent.fake_services,
                    dim_name="host",
                    dim_value="testhost",
                    prop_name="sf_services",
                    prop_values=[f"myapp-{i}", f"file-server-{i}"]
                )
            ), "Didn't get infrastructure correlation properties"

        time.sleep(10)
        agent.fake_services.reset_datapoints()

        assert ensure_never(
            p(has_datapoint, agent.fake_services, metric_name="sf.int.service.heartbeat"), timeout_seconds=5
        ), "Got infra correlation metric when it should have been expired"

        for i in range(0, 100):
            assert wait_for(
                p(
                    not_has_dim_set_prop,
                    agent.fake_services,
                    dim_name="host",
                    dim_value="testhost",
                    prop_name="sf_services",
                    prop_values=[f"myapp-{i}", f"file-server-{i}"]
                ),
                timeout_seconds=1
            ), "Got infra correlation property when it should have been expired"


def test_tracing_tags():
    """
    Test that the writer adds global dimensions as span tags when specified.
    """
    port = random.randint(5001, 20000)
    with Agent.run(
        dedent(
            f"""
        hostname: "testhost"
        globalDimensions:
          env: test
          os: linux
        writer:
            addGlobalDimensionsAsSpanTags: true
        monitors:
          - type: trace-forwarder
            listenAddress: localhost:{port}
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

        # It should keep the "env" tag from the original span and not overwrite
        # it.
        assert wait_for(
            p(has_trace_span, agent.fake_services, name="get", tags={"os": "linux", "env": "prod"})
        ), "Didn't get 'get' span"

        assert wait_for(
            p(
                has_trace_span,
                agent.fake_services,
                name="fetch",
                tags={"os": "linux", "env": "prod", "file": "test.pdf"},
            )
        ), "Didn't get fetch span"


def test_tracing_environment():
    """
    Test that the writer adds environment property from config or from span tags
    """
    port = random.randint(5001, 20000)
    with Agent.run(
            dedent(
                f"""
        hostname: "testhost"
        environment: "integ"
        globalDimensions:
          env: test
          os: linux
        writer:
            addGlobalDimensionsAsSpanTags: true
        monitors:
          - type: trace-forwarder
            listenAddress: localhost:{port}
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

        # Both environments (default from config and from span tag)
        # should be added as properties on the host dimension
        assert wait_for(
            p(
                has_dim_set_prop,
                agent.fake_services,
                dim_name="host",
                dim_value="testhost",
                prop_name="sf_environments",
                prop_values=["prod", "integ"]
            )
        ), "Didn't get infrastructure correlation properties"

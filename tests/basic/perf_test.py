# pylint: disable=too-many-locals
import json
import random
from functools import partial as p
from textwrap import dedent
import time
import requests

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_dim_set_prop, has_no_dim_set_prop, tcp_port_open_locally
from tests.helpers.util import retry_on_ebadf, wait_for

# mark all tests in this file as performance tests
pytestmark = [pytest.mark.perf_test]


def _test_span(service, environment):
    return [
        {
            "traceId": "0123456789abcdef",
            "name": "get",
            "id": _random_id(),
            "kind": "CLIENT",
            "timestamp": 1_538_406_065_536_000,
            "duration": 10000,
            "localEndpoint": {"serviceName": service, "ipv4": "10.0.0.1"},
            "tags": {"environment": environment},
        }
    ]


def _random_id():
    return "".join(random.choice("0123456789abcdef") for _ in range(16))


@retry_on_ebadf
def post_span(service_name, environment_name, port):
    return requests.post(
        f"http://localhost:{port}/v1/trace",
        headers={"Content-Type": "application/json"},
        data=json.dumps(_test_span(service_name, environment_name)),
    )


# pylint: disable=too-many-locals
def test_service_correlation():
    total_span_count = 25000
    environment_count = 25
    service_count = 5000
    expire_sec = 120

    port = random.randint(5001, 20000)
    with Agent.run(
        dedent(
            f"""
        hostname: "testhost"
        writer:
          maxRequests: 100
          traceHostCorrelationPurgeInterval: 1s
          traceHostCorrelationMetricsInterval: 1s
          sendTraceHostCorrelationMetrics: true
          staleServiceTimeout: {expire_sec}s
        monitors:
          - type: trace-forwarder
            listenAddress: localhost:{port}
        """
        ),
        profiling=True,
        debug=False,
        # This test generates a lot of spans and datapoints that we never check.
        # We must disable the storage of this data in the "fake" backend
        # or this test will consume a lot of (too much) memory.
        backend_options={"save_datapoints": False, "save_spans": False, "save_events": False},
    ) as agent:
        assert wait_for(p(tcp_port_open_locally, port)), "trace forwarder port never opened!"

        environment_names = {f"env-{e}" for e in range(0, environment_count)}
        service_names = {f"service-{s}" for s in range(0, service_count)}

        # send spans from a mixture of all environments/services
        for i in range(0, total_span_count):
            # rotate through the environment and service names
            environment_name = f"env-{i % environment_count}"
            service_name = f"service-{i % service_count}"
            resp = post_span(service_name, environment_name, port)
            assert resp.status_code == 200

        assert wait_for(
            p(
                has_dim_set_prop,
                agent.fake_services,
                dim_name="host",
                dim_value="testhost",
                prop_name="sf_services",
                prop_values=service_names,
            )
        ), "Didn't get all service name properties"

        assert wait_for(
            p(
                has_dim_set_prop,
                agent.fake_services,
                dim_name="host",
                dim_value="testhost",
                prop_name="sf_environments",
                prop_values=environment_names,
            )
        ), "Didn't get all environment name properties"

        # wait for services and environments to expire
        time.sleep(expire_sec)
        assert wait_for(
            p(
                has_no_dim_set_prop,
                agent.fake_services,
                dim_name="host",
                dim_value="testhost",
                prop_name="sf_services",
                prop_value=service_names,
            )
        ), "Didn't expire all services"

        assert wait_for(
            p(
                has_no_dim_set_prop,
                agent.fake_services,
                dim_name="host",
                dim_value="testhost",
                prop_name="sf_environments",
                prop_value=environment_names,
            )
        ), "Didn't expire all environments"

        agent.pprof_client.assert_goroutine_count_under(150)
        agent.pprof_client.assert_heap_alloc_under(200 * 1024 * 1024)


# pylint: disable=too-many-locals
def test_service_correlation_api_down():
    total_span_count = 25000
    environment_count = 25
    service_count = 5000
    expire_sec = 120
    send_delay = 0
    retries = 1

    port = random.randint(5001, 20000)
    with Agent.run(
        dedent(
            f"""
        hostname: "testhost"
        writer:
          maxRequests: 100
          traceHostCorrelationPurgeInterval: 1s
          traceHostCorrelationMetricsInterval: 1s
          sendTraceHostCorrelationMetrics: true
          staleServiceTimeout: {expire_sec}s
          traceHostCorrelationMaxRequestRetries: {retries}
          propertiesSendDelaySeconds: {send_delay}
        monitors:
          - type: trace-forwarder
            listenAddress: localhost:{port}
        """
        ),
        profiling=True,
        debug=False,
        # This test generates a lot of spans and datapoints that we never check.
        # We must disable the storage of this data in the "fake" backend
        # or this test will consume a lot of (too much) memory.
        backend_options={
            "save_datapoints": False,
            "save_spans": False,
            "save_events": False,
            "correlation_api_status_code": 500,
        },
    ) as agent:
        assert wait_for(p(tcp_port_open_locally, port)), "trace forwarder port never opened!"

        # send spans from a mixture of all environments/services
        for i in range(0, total_span_count):
            # rotate through the environment and service names
            environment_name = f"env-{i % environment_count}"
            service_name = f"service-{i % service_count}"
            resp = post_span(service_name, environment_name, port)
            assert resp.status_code == 200

        # get the peak heap size
        timeout_seconds = 120
        start = time.time()
        peak_heap = agent.pprof_client.get_heap_profile().total
        while True:
            time.sleep(10)
            assert time.time() - start < timeout_seconds
            new = agent.pprof_client.get_heap_profile().total
            print("old: {0} new: {1}".format(peak_heap, new))
            if new < peak_heap:
                break
            peak_heap = new

        # ensure our routine count isn't too high
        agent.pprof_client.assert_goroutine_count_under(150)

        # ensure the heap profile has come down some
        agent.pprof_client.assert_heap_alloc_under(peak_heap / 1.5)

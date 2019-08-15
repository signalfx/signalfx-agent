from functools import partial as p

import pytest
import requests
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for, wait_for_assertion
from tests.helpers.verify import verify

pytestmark = [pytest.mark.collectd, pytest.mark.haproxy, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("collectd/haproxy")

EXPECTED_DEFAULTS = METADATA.default_metrics


@pytest.mark.parametrize("version", ["1.9"])
def test_haproxy_basic(version):
    with run_service("haproxy", buildargs={"HAPROXY_VERSION": version}) as service_container:
        host = container_ip(service_container)
        assert wait_for(p(tcp_socket_open, host, 9000)), "haproxy not listening on port"

        with Agent.run(
            f"""
           monitors:
           - type: collectd/haproxy
             host: {host}
             port: 9000
             enhancedMetrics: false
           """
        ) as agent:
            requests.get(f"http://{host}:80", timeout=5)
            requests.get(f"http://{host}:80", timeout=5)
            verify(agent, EXPECTED_DEFAULTS, 10)


def test_haproxy_extra_metrics_enables_enhanced_metrics():
    with run_service("haproxy", buildargs={"HAPROXY_VERSION": "1.9"}) as service_container:
        host = container_ip(service_container)
        assert wait_for(p(tcp_socket_open, host, 9000)), "haproxy not listening on port"

        with Agent.run(
            f"""
           monitors:
           - type: collectd/haproxy
             host: {host}
             port: 9000
             extraMetrics:
              - gauge.tasks
           """
        ) as agent:
            target_metric = "gauge.tasks"
            assert target_metric in METADATA.nondefault_metrics

            def test():
                assert has_datapoint(agent.fake_services, metric_name=target_metric)

            wait_for_assertion(test)


def test_haproxy_enhanced_metrics_enables_filter_passthrough():
    with run_service("haproxy", buildargs={"HAPROXY_VERSION": "1.9"}) as service_container:
        host = container_ip(service_container)
        assert wait_for(p(tcp_socket_open, host, 9000)), "haproxy not listening on port"

        with Agent.run(
            f"""
           monitors:
           - type: collectd/haproxy
             host: {host}
             port: 9000
             enhancedMetrics: true
           """
        ) as agent:
            target_metric = "gauge.tasks"
            assert target_metric in METADATA.nondefault_metrics

            def test():
                assert has_datapoint(agent.fake_services, metric_name=target_metric)

            wait_for_assertion(test)

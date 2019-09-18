from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import datapoints_have_some_or_all_dims, has_log_message, any_metric_found
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, ensure_always
from tests.helpers.verify import verify

pytestmark = [pytest.mark.haproxy, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("haproxy")

EXPECTED_DEFAULTS = METADATA.default_metrics

EXPECTED_DEFAULTS_FROM_SOCKET = {
    "haproxy_connection_rate_all",
    "haproxy_idle_percent",
    "haproxy_requests",
    "haproxy_session_rate_all",
}


@pytest.mark.parametrize("version", ["1.9"])
def test_haproxy_default_metrics_from_stats_page(version):
    with run_service("haproxy", buildargs={"HAPROXY_VERSION": version}) as service_container:
        host = container_ip(service_container)
        with Agent.run(
            f"""
           monitors:
           - type: haproxy
             url: http://{host}:8080/stats?stats;csv
           """
        ) as agent:

            verify(agent, EXPECTED_DEFAULTS - EXPECTED_DEFAULTS_FROM_SOCKET, 10)
            assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


@pytest.mark.parametrize("version", ["1.9"])
def test_haproxy_default_metrics_from_stats_page_proxies_to_monitor_frontend_200s(version):
    with run_service("haproxy", buildargs={"HAPROXY_VERSION": version}) as service_container:
        host = container_ip(service_container)
        with Agent.run(
            f"""
           monitors:
           - type: haproxy
             url: http://{host}:8080/stats?stats;csv
             proxies: ["FRONTEND", "200s"]
           """
        ) as agent:
            assert ensure_always(
                p(
                    datapoints_have_some_or_all_dims,
                    agent.fake_services,
                    {"proxy_name": "200s", "service_name": "FRONTEND"},
                ),
                10,
            )
            assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"
            assert any_metric_found(agent.fake_services, ["haproxy_response_2xx"])


@pytest.mark.parametrize("version", ["1.9"])
def test_haproxy_default_metrics_from_stats_page_basic_auth(version):
    with run_service("haproxy", buildargs={"HAPROXY_VERSION": version}) as service_container:
        host = container_ip(service_container)
        with Agent.run(
            f"""
           monitors:
           - type: haproxy
             username: a_username
             password: a_password
             url: http://{host}:8081/stats?stats;csv
             proxies: ["FRONTEND", "200s"]
           """
        ) as agent:
            assert ensure_always(
                p(
                    datapoints_have_some_or_all_dims,
                    agent.fake_services,
                    {"proxy_name": "200s", "service_name": "FRONTEND"},
                ),
                10,
            )
            assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"
            assert any_metric_found(agent.fake_services, ["haproxy_response_2xx"])


@pytest.mark.parametrize("version", ["1.9"])
def test_haproxy_default_metrics_from_stats_page_basic_auth_wrong_password(version):
    with run_service("haproxy", buildargs={"HAPROXY_VERSION": version}) as service_container:
        host = container_ip(service_container)
        url = f"http://{host}:8081/stats?stats;csv"
        with Agent.run(
            f"""
           monitors:
           - type: haproxy
             username: a_username
             password: a_wrong_password
             url: {url}
             proxies: ["FRONTEND", "200s"]
           """
        ) as agent:
            assert ensure_always(
                p(
                    datapoints_have_some_or_all_dims,
                    agent.fake_services,
                    {"proxy_name": "200s", "service_name": "FRONTEND"},
                ),
                10,
            )
            assert has_log_message(agent.output.lower(), "error"), "error found in agent output!"

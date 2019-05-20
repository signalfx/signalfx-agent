from contextlib import contextmanager
from functools import partial as p
from pathlib import Path

import pytest
from requests import get, post

from tests.helpers.agent import Agent
from tests.helpers.assertions import container_cmd_exit_0, has_datapoint_with_dim, http_status, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_container, run_service, wait_for, retry
from tests.helpers.verify import verify

LATEST = "1.0.0-centos"
VERSIONS = ["0.13-centos", "0.14-centos", "0.15-centos", LATEST]

pytestmark = [
    pytest.mark.flaky(reruns=2, reruns_delay=5),
    pytest.mark.collectd,
    pytest.mark.kong,
    pytest.mark.monitor_with_endpoints,
]

METADATA = Metadata.from_package("collectd/kong")
SCRIPT_DIR = Path(__file__).parent

# Setup taken from https://github.com/signalfx/collectd-kong/blob/master/test/integration/test_e2e.py


def configure_kong(kong_admin, kong_version, echo):
    object_ids = set()
    service_paths = []
    if kong_version >= "0.13-centos":
        service_paths = ["sOne", "sTwo", "sThree"]
        for service_path in service_paths:
            service = post(kong_admin + "/services", json=dict(name=service_path, url=f"http://{echo}:8080/echo"))
            assert service.status_code == 201
            object_ids.add(service.json()["id"])
            route = post(
                kong_admin + "/routes", json=dict(service=dict(id=service.json()["id"]), paths=["/" + service_path])
            )
            assert route.status_code == 201
            object_ids.add(route.json()["id"])

    api_paths = []
    if kong_version < "1.0":
        api_paths = ["aOne", "aTwo", "aThree"]
        for api_path in api_paths:
            api = post(
                kong_admin + "/apis",
                json=dict(name=api_path, uris=["/" + api_path], upstream_url=f"http://{echo}:8080/echo"),
            )
            assert api.status_code == 201
            object_ids.add(api.json()["id"])

    kong_plugins = kong_admin + "/plugins"
    enable = post(kong_plugins, json=dict(name="signalfx"))
    assert enable.status_code == 201
    return service_paths + api_paths, object_ids


def run_traffic(paths, kong_proxy):
    status_codes = set()
    for _ in range(10):
        for path in paths:
            resp = get(f"{kong_proxy}/{path}")
            if resp.status_code != 204:
                assert b"headers:" in resp.content
            status_codes.add(str(resp.status_code))
    return status_codes


@contextmanager
def run_kong(kong_version):
    pg_env = dict(POSTGRES_USER="kong", POSTGRES_PASSWORD="kong", POSTGRES_DB="kong")
    kong_env = dict(
        KONG_ADMIN_LISTEN="0.0.0.0:8001",
        KONG_LOG_LEVEL="warn",
        KONG_DATABASE="postgres",
        KONG_PG_DATABASE=pg_env["POSTGRES_DB"],
        KONG_PG_PASSWORD=pg_env["POSTGRES_PASSWORD"],
    )

    with run_container("postgres:9.5", environment=pg_env) as db:
        db_ip = container_ip(db)
        kong_env["KONG_PG_HOST"] = db_ip

        assert wait_for(p(tcp_socket_open, db_ip, 5432))

        with run_service(
            "kong",
            name="kong-boot",
            buildargs={"KONG_VERSION": kong_version},
            environment=kong_env,
            command="sleep inf",
        ) as migrations:
            if kong_version in ["0.15-centos", "1.0.0-centos"]:
                assert container_cmd_exit_0(migrations, "kong migrations bootstrap")
            else:
                assert container_cmd_exit_0(migrations, "kong migrations up")

        with run_service(
            "kong", name="kong", buildargs={"KONG_VERSION": kong_version}, environment=kong_env
        ) as kong, run_container(
            "openresty/openresty:centos", files=[(SCRIPT_DIR / "echo.conf", "/etc/nginx/conf.d/echo.conf")]
        ) as echo:
            kong_ip = container_ip(kong)
            kong_admin = f"http://{kong_ip}:8001"
            assert wait_for(p(http_status, url=f"{kong_admin}/signalfx", status=[200]))

            paths, _ = configure_kong(kong_admin, kong_version, container_ip(echo))
            # Needs time to settle after creating routes.
            retry(lambda: run_traffic(paths, f"http://{kong_ip}:8000"), AssertionError, interval_seconds=2)
            yield kong_ip


@pytest.mark.parametrize("kong_version", VERSIONS)
def test_kong_included(kong_version):
    with run_kong(kong_version) as kong_ip:
        config = f"""
        monitors:
        - type: collectd/kong
          host: {kong_ip}
          port: 8001
        """

        with Agent.run(config) as agent:
            verify(agent, METADATA.included_metrics)
            assert has_datapoint_with_dim(agent.fake_services, "plugin", "kong"), "Didn't get Kong dimension"


@pytest.mark.parametrize("kong_version", VERSIONS)
def test_kong_all(kong_version):
    with run_kong(kong_version) as kong_ip:
        config = f"""
        monitors:
        - type: collectd/kong
          host: {kong_ip}
          port: 8001
          extraMetrics: ["*"]
        """

        with Agent.run(config) as agent:
            verify(agent, METADATA.all_metrics)
            assert has_datapoint_with_dim(agent.fake_services, "plugin", "kong"), "Didn't get Kong dimension"


def test_kong_extra_metric():
    """Test adding extra metric enables underlying config metric"""
    # counter.kong.connections.handled chosen because it's not reported by default by the monitor
    # and is not a default metric.
    with run_kong(LATEST) as kong_ip:
        config = f"""
        monitors:
        - type: collectd/kong
          host: {kong_ip}
          port: 8001
          extraMetrics:
          - counter.kong.connections.handled
        """

        with Agent.run(config) as agent:
            verify(agent, METADATA.included_metrics | {"counter.kong.connections.handled"})
            assert has_datapoint_with_dim(agent.fake_services, "plugin", "kong"), "Didn't get Kong dimension"


def test_kong_metric_config():
    """Test turning on metric config flag allows through filter"""
    with run_kong(LATEST) as kong_ip:
        config = f"""
        monitors:
        - type: collectd/kong
          host: {kong_ip}
          port: 8001
          metrics:
          - metric: connections_accepted
            report: true
        """
        with Agent.run(config) as agent:
            verify(agent, METADATA.included_metrics | {"counter.kong.connections.accepted"})
            assert has_datapoint_with_dim(agent.fake_services, "plugin", "kong"), "Didn't get Kong dimension"

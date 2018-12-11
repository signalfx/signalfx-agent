import os
import string
from functools import partial as p
from io import BytesIO
from textwrap import dedent

import pytest
from docker.errors import BuildError
from requests import RequestException, get

from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test
from tests.helpers.util import (
    container_ip,
    get_docker_client,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    retry,
    run_agent,
    run_container,
    wait_for,
)

pytestmark = [pytest.mark.collectd, pytest.mark.kong, pytest.mark.monitor_with_endpoints]


@pytest.fixture(scope="session")
def kong_image():
    dockerfile = BytesIO(
        bytes(
            dedent(
                r"""
        from kong:0.13-centos
        RUN yum install -y epel-release
        RUN yum install -y postgresql git
        WORKDIR /usr/local/share/lua/5.1/kong
        RUN sed -i '38ilua_shared_dict kong_signalfx_aggregation 10m;' templates/nginx_kong.lua
        RUN sed -i '38ilua_shared_dict kong_signalfx_locks 100k;' templates/nginx_kong.lua
        RUN sed -i '29i\ \ "signalfx",' constants.lua
        WORKDIR /opt/
        RUN git clone --depth 1 https://github.com/signalfx/kong-plugin-signalfx.git
        RUN cd kong-plugin-signalfx && luarocks make
        WORKDIR /
        RUN mkdir -p /usr/local/kong/logs
        RUN ln -s /dev/stderr /usr/local/kong/logs/error.log
        RUN ln -s /dev/stdout /usr/local/kong/logs/access.log
    """
            ),
            "ascii",
        )
    )
    client = get_docker_client()
    image, _ = retry(p(client.images.build, fileobj=dockerfile, forcerm=True), BuildError)
    try:
        yield image.short_id
    finally:
        client.images.remove(image=image.id, force=True)


@pytest.mark.flaky(reruns=2, reruns_delay=5)
def test_kong(kong_image):  # pylint: disable=redefined-outer-name
    kong_env = dict(
        KONG_ADMIN_LISTEN="0.0.0.0:8001", KONG_LOG_LEVEL="warn", KONG_DATABASE="postgres", KONG_PG_DATABASE="kong"
    )

    with run_container("postgres:9.5", environment=dict(POSTGRES_USER="kong", POSTGRES_DB="kong")) as db:
        db_ip = container_ip(db)
        kong_env["KONG_PG_HOST"] = db_ip

        def db_is_ready():
            return db.exec_run("pg_isready -U kong").exit_code == 0

        assert wait_for(db_is_ready)

        with run_container(kong_image, environment=kong_env, command="sleep inf") as migrations:

            def db_is_reachable():
                return migrations.exec_run("psql -h {} -U kong".format(db_ip)).exit_code == 0

            assert wait_for(db_is_reachable)
            assert migrations.exec_run("kong migrations up --v").exit_code == 0

        with run_container(kong_image, environment=kong_env) as kong:
            kong_ip = container_ip(kong)

            def kong_is_listening():
                try:
                    return get("http://{}:8001/signalfx".format(kong_ip)).status_code == 200
                except RequestException:
                    return False

            assert wait_for(kong_is_listening)

            config = string.Template(
                dedent(
                    """
            monitors:
              - type: collectd/kong
                host: $host
                port: 8001
                metrics:
                  - metric: connections_handled
                    report: true
            """
                )
            ).substitute(host=container_ip(kong))

            with run_agent(config) as [backend, _, _]:
                assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "kong")), "Didn't get Kong data point"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_kong_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = os.path.join(os.path.dirname(os.path.realpath(__file__)), "kong-k8s.yaml")
    dockerfile_dir = os.path.join(os.path.dirname(os.path.realpath(__file__)), "../../../test-services/kong")
    build_opts = {"tag": "kong:k8s-test"}
    minikube.build_image(dockerfile_dir, build_opts)
    monitors = [
        {"type": "collectd/kong", "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace)}
    ]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=[yaml],
        observer=k8s_observer,
        expected_metrics=get_monitor_metrics_from_selfdescribe(monitors[0]["type"]),
        expected_dims=get_monitor_dims_from_selfdescribe(monitors[0]["type"]),
        test_timeout=k8s_test_timeout,
    )

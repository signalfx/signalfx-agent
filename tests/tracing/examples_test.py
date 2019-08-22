import socket
from contextlib import contextmanager
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import container_cmd_exit_0, has_datapoint, has_trace_span, tcp_socket_open
from tests.helpers.util import container_ip, get_host_ip, run_container, run_service, wait_for

pytestmark = [pytest.mark.tracing, pytest.mark.tracing_example]

CONFIG = """
    hostname: "testhost"
    writer:
        traceHostCorrelationMetricsInterval: 1s
    monitors:
      - type: trace-forwarder
        listenAddress: {}:{}
"""

PYTHON_VERSIONS = ["2", "3"]
NODE_VERSIONS = ["8", "10", "12"]
GO_VERSIONS = ["1.10", "1.11", "1.12"]


def get_free_port():
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("", 0))
        return sock.getsockname()[1]


def assert_exec_run(cont, cmd):
    code, output = cont.exec_run(cmd)
    assert code == 0, "%s:\n%s" % (cmd, output.decode("utf-8"))
    return output.decode("utf-8")


@contextmanager
def run_agent():
    host_ip = get_host_ip()
    port = get_free_port()
    with Agent.run(CONFIG.format(host_ip, port), host=host_ip) as agent:
        assert wait_for(p(tcp_socket_open, host_ip, port)), "trace forwarder port never opened!"
        yield (agent, host_ip, port)


@pytest.mark.parametrize("python_version", PYTHON_VERSIONS, ids=["python%s" % v for v in PYTHON_VERSIONS])
def test_python_django(python_version):
    with run_agent() as [agent, host_ip, port]:
        service_name = "python-example"
        env = dict(SIGNALFX_ENDPOINT_URL=f"http://{host_ip}:{port}/v1/trace", SIGNALFX_SERVICE_NAME=service_name)
        with run_service(
            "tracing-examples/python-django", buildargs={"PYTHON_VERSION": python_version}, environment=env
        ) as django:
            assert wait_for(
                p(container_cmd_exit_0, django, "curl -f http://localhost:7000")
            ), "django service not started!"
            assert_exec_run(django, "python client.py --username testuser --password testing123 --port 7000")
            assert wait_for(
                p(has_trace_span, agent.fake_services, local_service_name=service_name, tags={"host": "testhost"})
            ), "Didn't get span tag"
            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="sf.int.service.heartbeat",
                    dimensions={"sf_hasService": service_name, "host": "testhost"},
                )
            ), "Didn't get host correlation datapoint"


@pytest.mark.parametrize("node_version", NODE_VERSIONS, ids=["node%s" % v for v in NODE_VERSIONS])
def test_nodejs_express(node_version):
    with run_agent() as [agent, host_ip, port]:
        service_name = "snowman-server"
        env = dict(SIGNALFX_INGEST_URL=f"http://{host_ip}:{port}/v1/trace")
        with run_service(
            "tracing-examples/nodejs-express", buildargs={"NODE_VERSION": node_version}, environment=env
        ) as express:
            assert wait_for(p(tcp_socket_open, container_ip(express), 3000)), "express service not started!"
            assert_exec_run(express, "npm run client new")
            assert_exec_run(express, "npm run client guess x")
            assert_exec_run(express, "npm run client answer")
            assert wait_for(
                p(has_trace_span, agent.fake_services, local_service_name=service_name, tags={"host": "testhost"})
            ), "Didn't get span tag"
            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="sf.int.service.heartbeat",
                    dimensions={"sf_hasService": service_name, "host": "testhost"},
                )
            ), "Didn't get host correlation datapoint"


@pytest.mark.parametrize("node_version", NODE_VERSIONS, ids=["node%s" % v for v in NODE_VERSIONS])
def test_nodejs_koa(node_version):
    with run_agent() as [agent, host_ip, port]:
        service_name = "wordExplorerServer"
        with run_container("mongo:3.6") as mongo:
            mongo_ip = container_ip(mongo)
            assert wait_for(p(tcp_socket_open, mongo_ip, 27017)), "mongo service not started!"
            env = dict(SIGNALFX_ENDPOINT_URL=f"http://{host_ip}:{port}/v1/trace", MONGOHOST=mongo_ip)
            with run_service(
                "tracing-examples/nodejs-koa", buildargs={"NODE_VERSION": node_version}, environment=env
            ) as koa:
                assert wait_for(p(tcp_socket_open, container_ip(koa), 3000)), "koa service not started!"
                assert_exec_run(koa, "npm run client explore create")
                assert wait_for(
                    p(has_trace_span, agent.fake_services, local_service_name=service_name, tags={"host": "testhost"})
                ), "Didn't get span tag"
                assert wait_for(
                    p(
                        has_datapoint,
                        agent.fake_services,
                        metric_name="sf.int.service.heartbeat",
                        dimensions={"sf_hasService": service_name, "host": "testhost"},
                    )
                ), "Didn't get host correlation datapoint"


@pytest.mark.parametrize("node_version", NODE_VERSIONS, ids=["node%s" % v for v in NODE_VERSIONS])
def test_nodejs_mongo(node_version):
    with run_agent() as [agent, host_ip, port]:
        service_name = "traced-mongo-logger"
        with run_container("mongo:3.6") as mongo:
            mongo_ip = container_ip(mongo)
            assert wait_for(p(tcp_socket_open, mongo_ip, 27017)), "mongo service not started!"
            env = dict(
                SIGNALFX_ENDPOINT_URL=f"http://{host_ip}:{port}/v1/trace",
                SIGNALFX_SERVICE_NAME=service_name,
                MONGOHOST=mongo_ip,
            )
            with run_service(
                "tracing-examples/nodejs-mongo", buildargs={"NODE_VERSION": node_version}, environment=env
            ) as mongo_logger:
                assert wait_for(p(tcp_socket_open, container_ip(mongo_logger), 8080)), "logger service not started!"
                assert_exec_run(mongo_logger, "bash -c \"echo -e 'foo\\n/q' | npm run client\"")
                assert wait_for(
                    p(has_trace_span, agent.fake_services, local_service_name=service_name, tags={"host": "testhost"})
                ), "Didn't get span tag"
                assert wait_for(
                    p(
                        has_datapoint,
                        agent.fake_services,
                        metric_name="sf.int.service.heartbeat",
                        dimensions={"sf_hasService": service_name, "host": "testhost"},
                    )
                ), "Didn't get host correlation datapoint"


@pytest.mark.parametrize("node_version", NODE_VERSIONS, ids=["node%s" % v for v in NODE_VERSIONS])
@pytest.mark.parametrize("mysql_client", ["1", "2"], ids=["mysql1", "mysql2"])
def test_nodejs_mysql(node_version, mysql_client):
    with run_agent() as [agent, host_ip, port]:
        service_name = "deedServer"
        mysql_env = dict(
            MYSQL_ROOT_PASSWORD="password", MYSQL_DATABASE="mysql_db", MYSQL_USER="admin", MYSQL_PASSWORD="password"
        )
        with run_container("mysql:5", environment=mysql_env) as mysql:
            mysql_ip = container_ip(mysql)
            assert wait_for(p(tcp_socket_open, mysql_ip, 3306), 60), "mysql service not started!"
            env = dict(
                SIGNALFX_ENDPOINT_URL=f"http://{host_ip}:{port}/v1/trace",
                DEEDSCHEDULER_MYSQL_CLIENT=mysql_client,
                MYSQLHOST=mysql_ip,
            )
            with run_service(
                "tracing-examples/nodejs-mysql", buildargs={"NODE_VERSION": node_version}, environment=env
            ) as deedserver:
                assert wait_for(p(tcp_socket_open, container_ip(deedserver), 3001)), "deedserver service not started!"
                assert_exec_run(deedserver, "npm run client add coding 'Create a TODO app' Sunday")
                assert_exec_run(deedserver, "npm run client list Sunday")
                assert wait_for(
                    p(has_trace_span, agent.fake_services, local_service_name=service_name, tags={"host": "testhost"})
                ), "Didn't get span tag"
                assert wait_for(
                    p(
                        has_datapoint,
                        agent.fake_services,
                        metric_name="sf.int.service.heartbeat",
                        dimensions={"sf_hasService": service_name, "host": "testhost"},
                    )
                ), "Didn't get host correlation datapoint"


@pytest.mark.parametrize("go_version", GO_VERSIONS, ids=["go%s" % v for v in GO_VERSIONS])
@pytest.mark.parametrize("mongo_driver", ["mgo", "mongo"])
def test_go_gin(go_version, mongo_driver):
    with run_agent() as [agent, host_ip, port]:
        service_name = "signalfx-battleship"
        with run_container("mongo:3.6") as mongo:
            mongo_ip = container_ip(mongo)
            assert wait_for(p(tcp_socket_open, mongo_ip, 27017)), "mongo service not started!"
            env = dict(
                TRACINGENDPOINT=f"http://{host_ip}:{port}/v1/trace", MONGOHOST=mongo_ip, MONGODRIVER=mongo_driver
            )
            with run_service("tracing-examples/go-gin", buildargs={"GO_VERSION": go_version}, environment=env) as gin:
                print(assert_exec_run(gin, "go version"))
                assert wait_for(p(tcp_socket_open, container_ip(gin), 3030)), "gin service not started!"
                assert_exec_run(gin, "go run ./player/player.go")
                assert wait_for(
                    p(has_trace_span, agent.fake_services, local_service_name=service_name, tags={"host": "testhost"})
                ), "Didn't get span tag"
                assert wait_for(
                    p(
                        has_datapoint,
                        agent.fake_services,
                        metric_name="sf.int.service.heartbeat",
                        dimensions={"sf_hasService": service_name, "host": "testhost"},
                    )
                ), "Didn't get host correlation datapoint"

from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, ensure_always, run_agent, run_service, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.postgresql, pytest.mark.monitor_with_endpoints]


METADATA = Metadata.from_package("postgresql")
ENV = ["POSTGRES_USER=test_user", "POSTGRES_PASSWORD=test_pwd", "POSTGRES_DB=postgres"]


@pytest.mark.parametrize("version", ["9.2-alpine", "9-alpine", "10-alpine", "11-alpine"])
def test_postgresql(version):
    with run_service(
        "postgres", buildargs={"POSTGRES_VERSION": version}, environment=ENV, print_logs=False
    ) as postgres_cont:
        host = container_ip(postgres_cont)
        assert wait_for(p(tcp_socket_open, host, 5432), 60), "service didn't start"

        with run_agent(
            dedent(
                f"""
                monitors:
                  - type: postgresql
                    host: {host}
                    port: 5432
                    connectionString: "user=test_user password=test_pwd dbname=postgres sslmode=disable"
                """
            )
        ) as [backend, _, _]:
            for metric in METADATA.included_metrics:
                assert wait_for(
                    p(has_datapoint, backend, metric_name=metric, dimensions={"database": "dvdrental"})
                ), f"Didn't get included postgresql metric {metric} for database dvdrental"

            assert wait_for(
                p(has_datapoint, backend, dimensions={"database": "postgres"})
            ), f"Didn't get metric for postgres default database"


def test_postgresql_database_filter():
    with run_service(
        "postgres", buildargs={"POSTGRES_VERSION": "11-alpine"}, environment=ENV, print_logs=False
    ) as postgres_cont:
        host = container_ip(postgres_cont)
        assert wait_for(p(tcp_socket_open, host, 5432), 60), "service didn't start"

        with run_agent(
            dedent(
                f"""
                monitors:
                  - type: postgresql
                    host: {host}
                    port: 5432
                    connectionString: "user=test_user password=test_pwd dbname=postgres sslmode=disable"
                    databases: ['*', '!postgres']
                """
            )
        ) as [backend, _, _]:
            for metric in METADATA.included_metrics:
                assert wait_for(
                    p(has_datapoint, backend, metric_name=metric, dimensions={"database": "dvdrental"})
                ), f"Didn't get included postgresql metric {metric} for database dvdrental"

            assert ensure_always(
                lambda: not has_datapoint(backend, dimensions={"database": "postgres"})
            ), f"Should not get metric for postgres default database"

from contextlib import contextmanager
from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.util import container_ip, run_container, run_service, wait_for

pytestmark = [pytest.mark.monitor_with_endpoints]


def test_sql_postgresql_db():
    with run_service(
        "postgres",
        buildargs={"POSTGRES_VERSION": "11-alpine"},
        environment=["POSTGRES_USER=test_user", "POSTGRES_PASSWORD=test_pwd", "POSTGRES_DB=postgres"],
    ) as postgres_cont:
        host = container_ip(postgres_cont)
        assert wait_for(p(tcp_socket_open, host, 5432), 60), "service didn't start"

        with Agent.run(
            dedent(
                f"""
                monitors:
                  - type: sql
                    host: {host}
                    port: 5432
                    dbDriver: postgres
                    params:
                      password: test_pwd
                      username: test_user
                    connectionString: >
                      user={{{{.username}}}} password={{{{.password}}}} dbname=dvdrental sslmode=disable
                      host={{{{.host}}}} port={{{{.port}}}}
                    queries:
                     - query: >
                         SELECT COUNT(*) as count, country FROM city
                         INNER JOIN country ON country.country_id=city.country_id
                         GROUP BY country;
                       metrics:
                        - metricName: cities_per_country
                          valueColumn: count
                          dimensionColumns: [country]
                """
            )
        ) as agent:
            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="cities_per_country",
                    dimensions={"country": "United States"},
                )
            ), f"Didn't get cities_per_country metric for USA"


@contextmanager
def run_mysql_replication():
    with run_container(
        "bitnami/mysql:8.0.20-debian-10-r61",
        environment=[
            "MYSQL_ROOT_PASSWORD=master_root_password",
            "MYSQL_REPLICATION_MODE=master",
            "MYSQL_REPLICATION_USER=my_repl_user",
            "MYSQL_REPLICATION_PASSWORD=my_repl_password",
            "MYSQL_USER=my_user",
            "MYSQL_PASSWORD=my_password",
            "MYSQL_DATABASE=my_database",
        ],
    ) as mysql_master:
        with run_container(
            "bitnami/mysql:8.0.20-debian-10-r61",
            environment=[
                "MYSQL_MASTER_ROOT_PASSWORD=master_root_password",
                "MYSQL_REPLICATION_MODE=slave",
                "MYSQL_REPLICATION_USER=my_repl_user",
                "MYSQL_REPLICATION_PASSWORD=my_repl_password",
                f"MYSQL_MASTER_HOST={container_ip(mysql_master)}",
            ],
        ) as mysql_slave:
            yield [container_ip(mysql_master), container_ip(mysql_slave)]


def test_sql_mysql_db():
    with run_mysql_replication() as [_, slave_ip]:
        with Agent.run(
            dedent(
                f"""
                monitors:
                  - type: sql
                    host: {slave_ip}
                    port: 3306
                    dbDriver: mysql
                    params:
                      username: root
                      password: master_root_password
                    connectionString: '{{{{.username}}}}:{{{{.password}}}}@tcp({{{{.host}}}})/mysql'
                    queries:
                     - query: 'SHOW SLAVE STATUS'
                       datapointExpressions:
                         - |
                             GAUGE("mysql.slave_sql_running",
                                   {{master_uuid: Master_UUID, channel: Channel_name}},
                                   Slave_SQL_Running == "Yes" ? 1 : 0)
                """
            )
        ) as agent:
            assert wait_for(
                p(has_datapoint, agent.fake_services, metric_name="mysql.slave_sql_running", value=1),
                timeout_seconds=120,
            ), f"Didn't get mysql.slave_sql_running metric"

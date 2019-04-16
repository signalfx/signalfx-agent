from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.util import container_ip, run_service, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.postgresql, pytest.mark.monitor_with_endpoints]


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

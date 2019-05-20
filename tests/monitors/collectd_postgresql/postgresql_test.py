from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_container, wait_for
from tests.helpers.verify import verify

pytestmark = [pytest.mark.collectd, pytest.mark.postgresql, pytest.mark.monitor_with_endpoints]

ENV = ["POSTGRES_USER=test_user", "POSTGRES_PASSWORD=test_pwd", "POSTGRES_DB=test"]

METADATA = Metadata.from_package("collectd/postgresql")


def test_postgresql_defaults():
    with run_container("postgres:10", environment=ENV) as cont:
        host = container_ip(cont)
        assert wait_for(p(tcp_socket_open, host, 5432), 60), "service didn't start"

        with Agent.run(
            f"""
                monitors:
                  - type: collectd/postgresql
                    host: {host}
                    port: 5432
                    username: "username1"
                    password: "password1"
                    queries:
                    - name: "exampleQuery"
                      minVersion: 60203
                      maxVersion: 200203
                      statement: |
                        SELECT coalesce(sum(n_live_tup), 0) AS live, coalesce(sum(n_dead_tup), 0)
                        AS dead FROM pg_stat_user_tables;
                      results:
                      - type: gauge
                        instancePrefix: live
                        valuesFrom:
                        - live
                    databases:
                    - name: test
                      username: "test_user"
                      password: "test_pwd"
                      interval: 5
                      expireDelay: 10
                      sslMode: disable
                """
        ) as agent:
            verify(agent, METADATA.included_metrics)


def test_postgresql_enhanced():
    with run_container("postgres:10", environment=ENV) as cont:
        host = container_ip(cont)
        assert wait_for(p(tcp_socket_open, host, 5432), 60), "service didn't start"

        target_metric = "pg_blks.toast_hit"
        assert target_metric in METADATA.nonincluded_metrics

        with Agent.run(
            f"""
                monitors:
                  - type: collectd/postgresql
                    host: {host}
                    port: 5432
                    extraMetrics:
                     - "{target_metric}"
                    username: "username1"
                    password: "password1"
                    databases:
                    - name: test
                      username: "test_user"
                      password: "test_pwd"
                      interval: 5
                      expireDelay: 10
                      sslMode: disable
                """
        ) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, metric_name="pg_blks.toast_hit"))

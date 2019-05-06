import string
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, has_datapoint_with_metric_name, tcp_socket_open
from tests.helpers.util import container_ip, run_container, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.postgresql, pytest.mark.monitor_with_endpoints]

CONFIG_TEMP = string.Template(
    """
monitors:
  - type: collectd/postgresql
    extraMetrics:
     - '*'
    host: $host
    port: 5432
    username: "username1"
    password: "password1"
    queries:
    - name: "exampleQuery"
      minVersion: 60203
      maxVersion: 200203
      statement: |
        SELECT coalesce(sum(n_live_tup), 0) AS live, coalesce(sum(n_dead_tup), 0) AS dead FROM pg_stat_user_tables;
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
)

ENV = ["POSTGRES_USER=test_user", "POSTGRES_PASSWORD=test_pwd", "POSTGRES_DB=test"]


def test_postgresql():
    with run_container("postgres:10", environment=ENV) as cont:
        host = container_ip(cont)
        config = CONFIG_TEMP.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 5432), 60), "service didn't start"

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "postgresql")
            ), "Didn't get postgresql datapoints"
            assert wait_for(p(has_datapoint_with_metric_name, agent.fake_services, "pg_blks.toast_hit"))
            import time

            time.sleep(5)

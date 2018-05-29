from functools import partial as p
import os
import string

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, run_agent, run_container, container_ip
from tests.helpers.assertions import *

config_temp = string.Template("""
monitors:
  - type: collectd/postgresql
    host: $host
    port: 5432
    username: "username1"
    password: "password1"
    queries:
    - name: "exampleQuery"
      minVersion: 60203
      maxVersion: 200203
      statement: "SELECT coalesce(sum(n_live_tup), 0) AS live, coalesce(sum(n_dead_tup), 0) AS dead FROM pg_stat_user_tables;"
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
""")

env = [
        "POSTGRES_USER=test_user",
        "POSTGRES_PASSWORD=test_pwd",
        "POSTGRES_DB=test"
      ]


def test_postgresql():
    with run_container("postgres:10", environment=env) as cont:
        host = container_ip(cont)
        config = config_temp.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 5432), 60), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "postgresql")), "Didn't get postgresql datapoints"
            assert wait_for(p(has_datapoint_with_metric_name, backend, "pg_blks.toast_hit"))

from functools import partial as p
import pytest
import string

from tests.helpers.util import wait_for, run_agent, run_container, container_ip
from tests.helpers.assertions import tcp_socket_open, has_datapoint_with_dim

pytestmark = [pytest.mark.collectd, pytest.mark.redis, pytest.mark.monitor_with_endpoints]

monitor_config = string.Template("""
monitors:
- type: telegraf/sqlserver
  host: $host
  port: 1433
  userID: sa
  password: P@ssw0rd!
  log: 0
""")


@pytest.mark.parametrize("image", [
    "microsoft/mssql-server-linux:2017-latest"
])
def test_redis(image):
    with run_container(image, environment={"ACCEPT_EULA":"Y", "MSSQL_PID": "Developer", "SA_PASSWORD": "P@ssw0rd!"}) as test_container:
        host = container_ip(test_container)
        config = monitor_config.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 1433), 60), "service not listening on port"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "sqlserver_database_io")), "didn't get database io datapoints"
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "sqlserver_waitstats")), "didn't get waitstats datapoints"
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "sqlserver_memory_clerks")), "didn't get datapoints"
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "sqlserver_performance")), "didn't get performance datapoints"
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "sqlserver_server_properties")), "didn't get performance datapoints"

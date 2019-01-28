from functools import partial as p
import pytest
import string

from helpers.assertions import has_any_metric_or_dim, has_log_message, tcp_socket_open
from helpers.util import (
    container_ip,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_agent,
    run_container,
    wait_for,
)

pytestmark = [pytest.mark.telegraf, pytest.mark.sqlserver, pytest.mark.monitor_with_endpoints]

monitor_config = string.Template(
    """
monitors:
- type: telegraf/sqlserver
  host: $host
  port: 1433
  userID: sa
  password: P@ssw0rd!
  log: 0
"""
)


@pytest.mark.parametrize("image", ["microsoft/mssql-server-linux:2017-latest"])
def test_sql(image):
    expected_metrics = get_monitor_metrics_from_selfdescribe("telegraf/sqlserver")
    expected_dims = get_monitor_dims_from_selfdescribe("telegraf/sqlserver")
    with run_container(
        image, environment={"ACCEPT_EULA": "Y", "MSSQL_PID": "Developer", "SA_PASSWORD": "P@ssw0rd!"}
    ) as test_container:
        host = container_ip(test_container)
        config = monitor_config.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 1433), 60), "service not listening on port"

        with run_agent(config) as [backend, get_output, _]:
            assert wait_for(
                p(has_any_metric_or_dim, backend, expected_metrics, expected_dims), timeout_seconds=60
            ), "timed out waiting for metrics and/or dimensions!"
            assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"

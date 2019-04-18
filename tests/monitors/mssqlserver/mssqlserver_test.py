import string
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_any_metric_or_dim, tcp_socket_open
from tests.helpers.util import (
    container_ip,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_container,
    wait_for,
)

pytestmark = [pytest.mark.telegraf, pytest.mark.sqlserver, pytest.mark.monitor_with_endpoints]

MONITOR_CONFIG = string.Template(
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
        config = MONITOR_CONFIG.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 1433), 60), "service not listening on port"

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_any_metric_or_dim, agent.fake_services, expected_metrics, expected_dims), timeout_seconds=60
            ), "timed out waiting for metrics and/or dimensions!"
            # TODO: there is a race that happens when the mssql user account can be logged in
            # need to find a way to verify that before running the agent because the monitor
            # reports a log in error once or twice before metrics report.  Once that is done
            # we should re-enable this check.
            # assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

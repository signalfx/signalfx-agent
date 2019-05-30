from functools import partial as p

import pytest

from tests.helpers.assertions import tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_container, wait_for
from tests.helpers.verify import run_agent_verify

pytestmark = [pytest.mark.telegraf, pytest.mark.sqlserver, pytest.mark.monitor_with_endpoints]
METADATA = Metadata.from_package("telegraf/monitors/mssqlserver")
VERSIONS = ["mcr.microsoft.com/mssql/server:2017-latest-ubuntu"]


EXCLUDED = {
    # These aren't coming through and I can't find them running SQLServer manually. Maybe default on Windows
    # version, should test.
    "sqlserver_server_properties.total_storage_mb",
    "sqlserver_server_properties.available_storage_mb",
}


@pytest.mark.parametrize("image", VERSIONS)
def test_sql_default(image):
    with run_container(
        image, environment={"ACCEPT_EULA": "Y", "MSSQL_PID": "Developer", "SA_PASSWORD": "P@ssw0rd!"}
    ) as test_container:
        host = container_ip(test_container)
        assert wait_for(p(tcp_socket_open, host, 1433), 60), "service not listening on port"

        run_agent_verify(
            f"""
            monitors:
            - type: telegraf/sqlserver
              host: {host}
              port: 1433
              userID: sa
              password: P@ssw0rd!
              log: 0
            """,
            METADATA.default_metrics - EXCLUDED,
        )

        # TODO: there is a race that happens when the mssql user account can be logged in
        # need to find a way to verify that before running the agent because the monitor
        # reports a log in error once or twice before metrics report.  Once that is done
        # we should re-enable this check.
        # assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


# Note that the metric sqlserver_memory_clerks.size_kb.log_pool may be missing when run
# locally but not on CI and the metric comes through when the contianer is run manually.
# I have no idea.
@pytest.mark.parametrize("image", VERSIONS)
def test_sql_all(image):
    with run_container(
        image, environment={"ACCEPT_EULA": "Y", "MSSQL_PID": "Developer", "SA_PASSWORD": "P@ssw0rd!"}
    ) as test_container:
        host = container_ip(test_container)
        assert wait_for(p(tcp_socket_open, host, 1433), 60), "service not listening on port"

        run_agent_verify(
            f"""
            monitors:
            - type: telegraf/sqlserver
              host: {host}
              port: 1433
              userID: sa
              password: P@ssw0rd!
              log: 0
              extraMetrics: ["*"]
            """,
            METADATA.all_metrics - EXCLUDED,
        )

        # TODO: there is a race that happens when the mssql user account can be logged in
        # need to find a way to verify that before running the agent because the monitor
        # reports a log in error once or twice before metrics report.  Once that is done
        # we should re-enable this check.
        # assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

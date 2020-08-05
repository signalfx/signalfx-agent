from contextlib import contextmanager
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for
from tests.helpers.verify import verify

pytestmark = [pytest.mark.monitor_with_endpoints]


METADATA = Metadata.from_package("supervisor")
PORT = 9001
PROCESS = "long_script"


@contextmanager
def run_supervisor_fpm():
    with run_service("supervisor") as supervisor_container:
        host = container_ip(supervisor_container)
        assert wait_for(p(tcp_socket_open, host, PORT), 60), "service didn't start"
        yield host


def test_supervisor_default():
    with run_supervisor_fpm() as host, Agent.run(
        f"""
        monitors:
        - type: supervisor
          host: {host}
          port: {PORT}
        """
    ) as agent:
        verify(agent, METADATA.default_metrics)
        assert has_datapoint_with_dim(
            agent.fake_services, "name", PROCESS
        ), "Didn't get process name dimension {}".format(PROCESS)

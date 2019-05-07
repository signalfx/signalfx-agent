import pytest

from tests.helpers.assertions import tcp_socket_open
from tests.helpers.util import run_service, container_ip, wait_for


@pytest.yield_fixture
def expvar_container_ip():
    """expvar container fixture"""
    with run_service("expvar") as container:
        host = container_ip(container)
        assert wait_for(lambda: tcp_socket_open(host, 8080), 60), "service didn't start"
        yield host

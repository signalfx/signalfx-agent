__all__ = ["expvar"]

import pytest

from tests.helpers.assertions import tcp_socket_open
from tests.helpers.util import run_service, container_ip, wait_for


@pytest.yield_fixture
def expvar():
    """expvar container fixture"""
    with run_service("expvar") as expvar_container:
        host = container_ip(expvar_container)
        assert wait_for(lambda: tcp_socket_open(host, 8080), 60), "service didn't start"
        yield host

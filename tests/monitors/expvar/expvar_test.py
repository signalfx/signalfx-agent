"""
Tests for the expvar monitor
"""

from contextlib import contextmanager

import pytest
from tests.helpers.assertions import tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for
from tests.helpers.verify import run_agent_verify_all_metrics, run_agent_verify, run_agent_verify_default_metrics

pytestmark = [pytest.mark.expvar, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("expvar")


@contextmanager
def run_expvar():
    """expvar container fixture"""
    with run_service("expvar") as container:
        host = container_ip(container)
        assert wait_for(lambda: tcp_socket_open(host, 8080), 60), "service didn't start"
        yield host


def test_expvar_default():
    with run_expvar() as expvar_container_ip:
        run_agent_verify_default_metrics(
            f"""
            monitors:
            - type: expvar
              host: {expvar_container_ip}
              port: 8080
            """,
            METADATA,
        )


def test_expvar_enhanced():
    with run_expvar() as expvar_container_ip:
        run_agent_verify_all_metrics(
            f"""
            monitors:
            - type: expvar
              host: {expvar_container_ip}
              port: 8080
              enhancedMetrics: true
            """,
            METADATA,
        )


def test_expvar_custom_metric():
    expected = METADATA.default_metrics | {"queues.count"}
    with run_expvar() as expvar_container_ip:
        run_agent_verify(
            f"""
            monitors:
            - type: expvar
              host: {expvar_container_ip}
              port: 8080
              metrics:
              - JSONPath: queues.count
                type: gauge
            """,
            expected,
        )

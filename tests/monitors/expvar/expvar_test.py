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
    """
    Given the JSON object below:
    {
        "queues": {
            "count": 5,
            "lengths": [ 4, 2, 1, 0, 5]
        }
    }
    """
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


def test_expvar_path_separator():
    """
    Given the JSON object below:
    {
        "queues": {
            "count": 5,
            "lengths": [ 4, 2, 1, 0, 5]
        }
    }
    """
    expected = METADATA.default_metrics | {"queues.count"}
    with run_expvar() as expvar_container_ip:
        run_agent_verify(
            f"""
            monitors:
            - type: expvar
              host: {expvar_container_ip}
              port: 8080
              metrics:
              - JSONPath: queues/count
                pathSeparator: /
                type: gauge
            """,
            expected,
        )


def test_expvar_escape_character():
    """
    Given the JSON object below, using the escape character '\' on the path separator character '.', '.' should be
    treated literally as part of the metric name.
    {
    ...
        "kafka.ex-jaeger-transaction.ok": 11
    ...
    }
    """
    expected = METADATA.default_metrics | {"kafka.ex-jaeger-transaction.ok"}
    with run_expvar() as expvar_container_ip:
        run_agent_verify(
            f"""
            monitors:
            - type: expvar
              host: {expvar_container_ip}
              port: 8080
              metrics:
              - JSONPath: 'kafka\\.ex-jaeger-transaction\\.ok'
                type: gauge
            """,
            expected,
        )


def test_expvar_one_or_more_digit_regex_json_path():
    """
    Given the JSON object below
    {
        "memory": {
                "Allocations": [
                    {"Size": 96, "Mallocs": 64, "Frees": 32},
                    {"Size": 32, "Mallocs": 16, "Frees": 16},
                    {"Size": 64, "Mallocs": 16, "Frees": 48}
                ]
                "HeapAllocation": 96
        }
    }
    """
    expected = METADATA.default_metrics | {"memory.allocations.mallocs"}
    with run_expvar() as expvar_container_ip:
        run_agent_verify(
            f"""
            monitors:
            - type: expvar
              host: {expvar_container_ip}
              port: 8080
              metrics:
              - JSONPath: 'memory.Allocations.\\\\d+.Mallocs'
                type: gauge
            """,
            expected,
        )


def test_expvar_one_or_more_character_regex_json_path():
    """
    Given the JSON object below
    {
        "memory": {
                "Allocations": [
                    {"Size": 96, "Mallocs": 64, "Frees": 32},
                    {"Size": 32, "Mallocs": 16, "Frees": 16},
                    {"Size": 64, "Mallocs": 16, "Frees": 48}
                ]
                "HeapAllocation": 96
        }
    }
    """
    expected = METADATA.default_metrics | {"memory.allocations.frees"}
    with run_expvar() as expvar_container_ip:
        run_agent_verify(
            f"""
            monitors:
            - type: expvar
              host: {expvar_container_ip}
              port: 8080
              metrics:
              - JSONPath: 'memory.Allocations.\\.+.Frees'
                type: gauge
            """,
            expected,
        )


def test_expvar_2_one_or_more_character_regex_json_path():
    """
    Given the JSON object below
    {
        "memory": {
                "Allocations": [
                    {"Size": 96, "Mallocs": 64, "Frees": 32},
                    {"Size": 32, "Mallocs": 16, "Frees": 16},
                    {"Size": 64, "Mallocs": 16, "Frees": 48}
                ]
                "HeapAllocation": 96
        }
    }
    """
    expected = METADATA.default_metrics | {
        "memory.allocations.size",
        "memory.allocations.mallocs",
        "memory.allocations.frees",
    }
    with run_expvar() as expvar_container_ip:
        run_agent_verify(
            f"""
            monitors:
            - type: expvar
              host: {expvar_container_ip}
              port: 8080
              metrics:
              - JSONPath: 'memory.Allocations.\\.+.\\.+'
                type: gauge
            """,
            expected,
        )


def test_expvar_empty_object_for_metric_value():
    """
    Given the JSON object below
    {
        "willplayad.in_flight": 0,
        "willplayad.response.noserv": {},
        "willplayad.response.serv": 0,
        "willplayad.start": 0
    }
    """
    expected = METADATA.default_metrics | {
        "willplayad.in_flight",
        # "willplayad.response.noserv",
        "willplayad.response.serv",
        "willplayad.start",
    }
    with run_expvar() as expvar_container_ip:
        run_agent_verify(
            f"""
            monitors:
            - type: expvar
              host: {expvar_container_ip}
              port: 8080
              metrics:
              - JSONPath: 'willplay*'
                pathSeparator: /
                type: gauge
            """,
            expected,
        )

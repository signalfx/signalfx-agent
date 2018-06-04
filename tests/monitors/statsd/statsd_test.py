"""
Tests the collectd/statsd monitor
"""
import os
import pytest
import socket

from functools import partial as p
from tests.helpers.util import wait_for, run_agent
from tests.helpers.assertions import (
    has_datapoint_with_dim,
    has_datapoint_with_metric_name,
    udp_port_open_locally
)
from tests.kubernetes.utils import run_k8s_monitors_test

pytestmark = [pytest.mark.collectd, pytest.mark.statsd, pytest.mark.monitor_without_endpoints]


def send_udp_message(host, port, msg):
    """
    Send a datagram to the given host/port
    """
    sock = socket.socket(socket.AF_INET, # Internet
                         socket.SOCK_DGRAM) # UDP
    sock.sendto(msg.encode("utf-8"), (host, port))


def test_statsd_monitor():
    """
    Test basic functionality
    """
    with run_agent("""
monitors:
  - type: collectd/statsd
    listenAddress: localhost
    listenPort: 8125
    counterSum: true
""") as [backend, _, _]:
        assert wait_for(p(udp_port_open_locally, 8125)), "statsd port never opened!"
        send_udp_message("localhost", 8125, "statsd.[foo=bar,dim=val]test:1|g")

        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "statsd")),\
            "Didn't get statsd datapoints"
        assert wait_for(p(has_datapoint_with_metric_name, backend, "gauge.statsd.test")),\
            "Didn't get statsd.test metric"
        assert wait_for(p(has_datapoint_with_dim, backend, "foo", "bar")),\
            "Didn't get foo dimension"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_statsd_in_k8s(agent_image, minikube, k8s_test_timeout, k8s_namespace):
    monitors = [
        {"type": "collectd/statsd",
         "listenAddress": "127.0.0.1",
         "listenPort": 8125,
         "counterSum": True,
         "deleteSets": True,
         "deleteCounters": True,
         "deleteTimers": True,
         "deleteGauges": True},
    ]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        expected_metrics={"gauge.statsd.test"},
        expected_dims={"foo", "dim"},
        test_timeout=k8s_test_timeout)


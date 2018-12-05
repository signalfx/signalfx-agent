"""
Tests the collectd/statsd monitor
"""
from functools import partial as p

import pytest

from helpers.assertions import has_datapoint_with_dim, has_datapoint_with_metric_name, udp_port_open_locally
from helpers.kubernetes.utils import run_k8s_monitors_test
from helpers.util import run_agent, send_udp_message, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.statsd, pytest.mark.monitor_without_endpoints]


def test_statsd_monitor():
    """
    Test basic functionality
    """
    with run_agent(
        """
monitors:
  - type: collectd/statsd
    listenAddress: localhost
    listenPort: 8125
    counterSum: true
"""
    ) as [backend, _, _]:
        assert wait_for(p(udp_port_open_locally, 8125)), "statsd port never opened!"
        send_udp_message("localhost", 8125, "statsd.[foo=bar,dim=val]test:1|g")

        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "statsd")), "Didn't get statsd datapoints"
        assert wait_for(
            p(has_datapoint_with_metric_name, backend, "gauge.statsd.test")
        ), "Didn't get statsd.test metric"
        assert wait_for(p(has_datapoint_with_dim, backend, "foo", "bar")), "Didn't get foo dimension"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_statsd_in_k8s(agent_image, minikube, k8s_test_timeout, k8s_namespace):
    # hack to populate data for statsd
    minikube.container.exec_run(
        [
            "/bin/bash",
            "-c",
            'n=0; while [ $n -le %d ]; do \
            echo "statsd.[foo=bar,dim=val]test:1|g" | nc -w 1 -u 127.0.0.1 8125; \
            sleep 1; \
            (( n += 1 )); \
            done'
            % k8s_test_timeout,
        ],
        detach=True,
    )
    monitors = [
        {
            "type": "collectd/statsd",
            "listenAddress": "127.0.0.1",
            "listenPort": 8125,
            "counterSum": True,
            "deleteSets": True,
            "deleteCounters": True,
            "deleteTimers": True,
            "deleteGauges": True,
        }
    ]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        expected_metrics={"gauge.statsd.test"},
        expected_dims={"foo", "dim"},
        test_timeout=k8s_test_timeout,
    )

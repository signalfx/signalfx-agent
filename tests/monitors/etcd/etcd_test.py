import string
from functools import partial as p
from pathlib import Path

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test
from tests.helpers.util import (
    container_ip,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_container,
    wait_for,
)

pytestmark = [pytest.mark.collectd, pytest.mark.etcd, pytest.mark.monitor_with_endpoints]

SCRIPT_DIR = Path(__file__).parent.resolve()

ETCD_COMMAND = (
    "-listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 "
    "-advertise-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001"
)

ETCD_CONFIG = string.Template(
    """
monitors:
  - type: collectd/etcd
    host: $host
    port: 2379
    clusterName: test-cluster
"""
)


def test_etcd_monitor():
    with run_container("quay.io/coreos/etcd:v2.3.8", command=ETCD_COMMAND) as etcd_cont:
        host = container_ip(etcd_cont)
        config = ETCD_CONFIG.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 2379), 60), "service didn't start"

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "etcd")
            ), "Didn't get etcd datapoints"


@pytest.mark.kubernetes
def test_etcd_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = SCRIPT_DIR / "etcd-k8s.yaml"
    monitors = [
        {
            "type": "collectd/etcd",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
            "clusterName": "etcd-cluster",
            "skipSSLValidation": True,
            "enhancedMetrics": True,
        }
    ]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=[yaml],
        observer=k8s_observer,
        expected_metrics=get_monitor_metrics_from_selfdescribe(monitors[0]["type"]),
        expected_dims=get_monitor_dims_from_selfdescribe(monitors[0]["type"]),
        test_timeout=k8s_test_timeout,
    )

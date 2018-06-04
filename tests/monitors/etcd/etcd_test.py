from functools import partial as p
import os
import pytest
import string

from tests.helpers.util import wait_for, run_agent, run_container, container_ip
from tests.helpers.assertions import *
from tests.kubernetes.utils import (
    run_k8s_monitors_test,
    get_metrics_from_doc,
    get_dims_from_doc,
)

pytestmark = [pytest.mark.collectd, pytest.mark.etcd, pytest.mark.monitor_with_endpoints]

ETCD_COMMAND="-listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 -advertise-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001"

etcd_config = string.Template("""
monitors:
  - type: collectd/etcd
    host: $host
    port: 2379
    clusterName: test-cluster
""")


def test_etcd_monitor():
    with run_container("quay.io/coreos/etcd:v2.3.8", command=ETCD_COMMAND) as etcd_cont:
        host = container_ip(etcd_cont)
        config = etcd_config.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 2379), 60), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "etcd")), "Didn't get etcd datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_etcd_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    monitors = [
        {"type": "collectd/etcd",
         "discoveryRule": 'container_image =~ "etcd" && private_port == 2379 && kubernetes_namespace == "%s"' % k8s_namespace,
         "clusterName": "etcd-cluster",
         "skipSSLValidation": True,
         "enhancedMetrics": True},
    ]
    yamls = [os.path.join(os.path.dirname(os.path.realpath(__file__)), "etcd-k8s.yaml")]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=yamls,
        observer=k8s_observer,
        expected_metrics=get_metrics_from_doc("collectd-etcd.md"),
        expected_dims=get_dims_from_doc("collectd-etcd.md"),
        test_timeout=k8s_test_timeout)


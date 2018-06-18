from functools import partial as p
import os
import pytest
import string

from tests.helpers.util import wait_for, run_agent, run_container, container_ip
from tests.helpers.assertions import tcp_socket_open, has_datapoint_with_dim, http_status
from tests.kubernetes.utils import (
    run_k8s_monitors_test,
    get_metrics_from_doc,
    get_dims_from_doc,
    get_discovery_rule,
)

pytestmark = [pytest.mark.collectd, pytest.mark.marathon, pytest.mark.monitor_with_endpoints]

monitor_config = string.Template("""
monitors:
- type: collectd/marathon
  host: $host
  port: 8080
""")


@pytest.mark.parametrize("marathon_image", [
    "mesosphere/marathon:v1.1.1",
    "mesosphere/marathon:v1.6.352"
])
def test_marathon(marathon_image):
    with run_container("zookeeper:3.5") as zookeeper:
        zkhost = container_ip(zookeeper)
        assert wait_for(p(tcp_socket_open, zkhost, 2181), 60), "zookeeper didn't start"
        with run_container(marathon_image,
                           command=["--master", "localhost:5050", "--zk", "zk://{0}:2181/marathon".format(zkhost)]
                           ) as service_container:
            host = container_ip(service_container)
            config = monitor_config.substitute(host=host)
            assert wait_for(p(tcp_socket_open, host, 8080), 120), "marathon not listening on port"
            assert wait_for(p(http_status, url="http://{0}:8080/v2/info".format(host), status=[200]), 120), "service didn't start"

            with run_agent(config) as [backend, _, _]:
                assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "marathon")), "didn't get datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_marathon_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = os.path.join(os.path.dirname(os.path.realpath(__file__)), "marathon-k8s.yaml")
    monitors = [
        {"type": "collectd/marathon",
         "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
         "username": "testuser", "password": "testing123"},
    ]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=[yaml],
        observer=k8s_observer,
        expected_metrics=get_metrics_from_doc("collectd-marathon.md"),
        expected_dims=get_dims_from_doc("collectd-marathon.md"),
        test_timeout=k8s_test_timeout)


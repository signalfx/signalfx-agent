from functools import partial as p
import os
import pytest
import string

from tests.helpers.util import wait_for, run_agent, run_service, container_ip
from tests.helpers.assertions import *
from tests.kubernetes.utils import (
    run_k8s_monitors_test,
    get_metrics_from_doc,
    get_dims_from_doc,
)

pytestmark = [pytest.mark.collectd, pytest.mark.cassandra, pytest.mark.monitor_with_endpoints]


cassandra_config = string.Template("""
monitors:
  - type: collectd/cassandra
    host: $host
    port: 7199
    username: cassandra
    password: cassandra
""")

def test_cassandra():
    with run_service("cassandra") as cassandra_cont:
        config = cassandra_config.substitute(host=container_ip(cassandra_cont))

        # Wait for the JMX port to be open in the container
        assert wait_for(p(container_cmd_exit_0, cassandra_cont,
                        "sh -c 'cat /proc/net/tcp | grep 1C1F'")), "Cassandra JMX didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_metric_name, backend, "counter.cassandra.ClientRequest.Read.Latency.Count"), 30), "Didn't get Cassandra datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_cassandra_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    monitors = [
        {"type": "collectd/cassandra",
         "discoveryRule": 'container_image =~ "cassandra" && private_port == 7199 && kubernetes_namespace == "%s"' % k8s_namespace,
         "username": "testuser", "password": "testing123"}
    ]
    yamls = [os.path.join(os.path.dirname(os.path.realpath(__file__)), "cassandra-k8s.yaml")]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=yamls,
        observer=k8s_observer,
        expected_metrics=get_metrics_from_doc("collectd-cassandra.md"),
        expected_dims=get_dims_from_doc("collectd-cassandra.md"),
        test_timeout=k8s_test_timeout)


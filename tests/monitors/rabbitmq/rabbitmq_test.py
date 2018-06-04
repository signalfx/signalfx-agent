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

pytestmark = [pytest.mark.collectd, pytest.mark.rabbitmq, pytest.mark.monitor_with_endpoints]


rabbitmq_config = string.Template("""
monitors:
  - type: collectd/rabbitmq
    host: $host
    port: 15672
    username: guest
    password: guest
    collectNodes: true
    collectChannels: true
""")


def test_rabbitmq():
    with run_container("rabbitmq:3.6-management") as rabbitmq_cont:
        host = container_ip(rabbitmq_cont)
        config = rabbitmq_config.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 15672), 60), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "rabbitmq")), "Didn't get rabbitmq datapoints"
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin_instance", "%s-15672" % host)), \
                "Didn't get expected plugin_instance dimension"


def test_rabbitmq_broker_name():
    with run_container("rabbitmq:3.6-management") as rabbitmq_cont:
        host = container_ip(rabbitmq_cont)
        config = rabbitmq_config.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 15672), 60), "service didn't start"

        with run_agent("""
monitors:
  - type: collectd/rabbitmq
    host: %s
    brokerName: '{{.host}}-{{.username}}'
    port: 15672
    username: guest
    password: guest
    collectNodes: true
    collectChannels: true
        """ % (host,)) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin_instance", "%s-guest" % host)), \
                "Didn't get expected plugin_instance dimension"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_rabbitmq_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    monitors = [
        {"type": "collectd/rabbitmq",
         "discoveryRule": 'container_image =~ "rabbitmq" && private_port == 15672 && kubernetes_namespace == "%s"' % k8s_namespace,
         "collectChannels": True,
         "collectConnections": True,
         "collectExchanges": True,
         "collectNodes": True,
         "collectQueues": True,
         "username": "testuser", "password": "testing123"},
    ]
    yamls = [os.path.join(os.path.dirname(os.path.realpath(__file__)), "rabbitmq-k8s.yaml")]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=yamls,
        observer=k8s_observer,
        expected_metrics=get_metrics_from_doc("collectd-rabbitmq.md"),
        expected_dims=get_dims_from_doc("collectd-rabbitmq.md"),
        test_timeout=k8s_test_timeout)


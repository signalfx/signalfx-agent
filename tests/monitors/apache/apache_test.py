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

pytestmark = [pytest.mark.collectd, pytest.mark.apache, pytest.mark.monitor_with_endpoints]

apache_config = string.Template("""
monitors:
  - type: collectd/apache
    host: $host
    port: 80
""")


def test_apache():
    with run_service("apache") as apache_container:
        host = container_ip(apache_container)
        config = apache_config.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 80), 60), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "apache")), "Didn't get apache datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_apache_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    monitors = [
        {"type": "collectd/apache",
         "discoveryRule": 'container_image =~ "httpd" && private_port == 80 && kubernetes_namespace == "%s"' % k8s_namespace,
         "url": 'http://{{.Host}}:{{.Port}}/mod_status?auto',
         "username": "testuser", "password": "testing123"}
    ]
    yamls = [os.path.join(os.path.dirname(os.path.realpath(__file__)), "apache-k8s.yaml")]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=yamls,
        observer=k8s_observer,
        expected_metrics=get_metrics_from_doc("collectd-apache.md"),
        expected_dims=get_dims_from_doc("collectd-apache.md"),
        test_timeout=k8s_test_timeout)


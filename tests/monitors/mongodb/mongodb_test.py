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
    get_discovery_rule,
)

pytestmark = [pytest.mark.collectd, pytest.mark.mongodb, pytest.mark.monitor_with_endpoints]

mongo_config = string.Template("""
monitors:
  - type: collectd/mongodb
    host: $host
    port: 27017
    databases: [admin]
""")


def test_mongo():
    with run_container("mongo:3.6") as mongo_cont:
        host = container_ip(mongo_cont)
        config = mongo_config.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 27017), 60), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "mongo")), "Didn't get mongo datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_mongodb_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = os.path.join(os.path.dirname(os.path.realpath(__file__)), "mongodb-k8s.yaml")
    monitors = [
        {"type": "collectd/mongodb",
         "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
         "databases": ["admin"],
         "username": "testuser", "password": "testing123"},
    ]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=[yaml],
        observer=k8s_observer,
        expected_metrics=get_metrics_from_doc("collectd-mongodb.md"),
        expected_dims=get_dims_from_doc("collectd-mongodb.md"),
        test_timeout=k8s_test_timeout)


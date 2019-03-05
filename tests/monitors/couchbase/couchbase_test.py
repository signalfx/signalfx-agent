import os
import string
from functools import partial as p

import pytest

from tests.helpers.assertions import has_datapoint_with_dim, http_status, tcp_socket_open
from tests.helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test
from tests.helpers.util import (
    container_ip,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_agent,
    run_service,
    wait_for,
)

pytestmark = [pytest.mark.collectd, pytest.mark.couchbase, pytest.mark.monitor_with_endpoints]

COUCHBASE_CONFIG = string.Template(
    """
monitors:
  - type: collectd/couchbase
    host: $host
    port: 8091
    collectTarget: NODE
    username: administrator
    password: password
"""
)


@pytest.mark.flaky(reruns=2)
@pytest.mark.parametrize("tag", ["enterprise-4.0.0", "enterprise-5.1.0"])
def test_couchbase(tag):
    with run_service(
        "couchbase", buildargs={"COUCHBASE_VERSION": tag}, hostname="node1.cluster"
    ) as couchbase_container:
        host = container_ip(couchbase_container)
        config = COUCHBASE_CONFIG.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 8091), 60), "service not listening on port"
        assert wait_for(
            p(http_status, url="http://{0}:8091/pools".format(host), status=[401]), 120
        ), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "couchbase")
            ), "Didn't get couchbase datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_couchbase_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = os.path.join(os.path.dirname(os.path.realpath(__file__)), "couchbase-k8s.yaml")
    build_opts = {"buildargs": {"COUCHBASE_VERSION": "enterprise-5.1.0"}, "tag": "couchbase:test"}
    minikube.build_image("couchbase", build_opts)
    monitors = [
        {
            "type": "collectd/couchbase",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
            "collectTarget": "NODE",
            "username": "administrator",
            "password": "testing123",
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

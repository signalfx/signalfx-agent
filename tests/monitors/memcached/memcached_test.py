import os
import pytest

from tests.kubernetes.utils import (
    run_k8s_monitors_test,
    get_metrics_from_doc,
    get_dims_from_doc,
)

pytestmark = [pytest.mark.collectd, pytest.mark.memcached, pytest.mark.monitor_with_endpoints]


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_memcached_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    monitors = [
        {"type": "collectd/memcached",
         "discoveryRule": 'container_image =~ "memcached" && private_port == 11211 && kubernetes_namespace == "%s"' % k8s_namespace},
    ]
    yamls = [os.path.join(os.path.dirname(os.path.realpath(__file__)), "memcached-k8s.yaml")]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=yamls,
        observer=k8s_observer,
        expected_metrics=get_metrics_from_doc("collectd-memcached.md"),
        expected_dims=get_dims_from_doc("collectd-memcached.md"),
        test_timeout=k8s_test_timeout)


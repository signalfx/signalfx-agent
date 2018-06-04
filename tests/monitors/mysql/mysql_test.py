import os
import pytest

from tests.kubernetes.utils import (
    run_k8s_monitors_test,
    get_metrics_from_doc,
    get_dims_from_doc,
)

pytestmark = [pytest.mark.collectd, pytest.mark.mysql, pytest.mark.monitor_with_endpoints]


@pytest.mark.k8s
@pytest.mark.kubernetes
@pytest.mark.parametrize(
    "k8s_yaml",
    ["mysql4-k8s.yaml", "mysql57-k8s.yaml", pytest.mark.skip("mysql8-k8s.yaml")],
    ids=["mysql4", "mysql5.7", "mysql8"])
def test_mysql_in_k8s(agent_image, minikube, k8s_observer, k8s_yaml, k8s_test_timeout, k8s_namespace):
    monitors = [
        {"type": "collectd/mysql",
         "discoveryRule": 'container_image =~ "mysql" && private_port == 3306 && kubernetes_namespace == "%s"' % k8s_namespace,
         "databases": [{"name": "mysql", "username": "root", "password": "testing123"}],
         "username": "root", "password": "testing123"},
    ]
    yamls = [os.path.join(os.path.dirname(os.path.realpath(__file__)), k8s_yaml)]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=yamls,
        observer=k8s_observer,
        expected_metrics=get_metrics_from_doc("collectd-mysql.md"),
        expected_dims=get_dims_from_doc("collectd-mysql.md"),
        test_timeout=k8s_test_timeout)


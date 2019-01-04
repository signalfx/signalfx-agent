import os
import pytest

from tests.helpers.util import get_monitor_metrics_from_selfdescribe, get_monitor_dims_from_selfdescribe
from tests.helpers.kubernetes.utils import run_k8s_monitors_test, get_discovery_rule

pytestmark = [pytest.mark.collectd, pytest.mark.spark, pytest.mark.monitor_with_endpoints]


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_spark_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = os.path.join(os.path.dirname(os.path.realpath(__file__)), "spark-k8s.yaml")
    monitors = [
        {
            "type": "collectd/spark",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
            "clusterType": "Standalone",
            "isMaster": True,
            "collectApplicationMetrics": True,
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

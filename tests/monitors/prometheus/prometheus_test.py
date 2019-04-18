from pathlib import Path

import pytest

from tests.helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test, get_metrics

pytestmark = [pytest.mark.prometheus, pytest.mark.monitor_with_endpoints]

DIR = Path(__file__).parent.resolve()


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_prometheus_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = DIR / "prometheus-k8s.yaml"
    monitors = [
        {
            "type": "prometheus-exporter",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
            "useHTTPS": False,
            "skipVerify": True,
            "metricPath": "/metrics",
        }
    ]
    expected_metrics = get_metrics(DIR)
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=[yaml],
        observer=k8s_observer,
        expected_metrics=expected_metrics,
        test_timeout=k8s_test_timeout,
    )

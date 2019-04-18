from pathlib import Path

import pytest

from tests.helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test, get_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.genericjmx, pytest.mark.monitor_with_endpoints]

DIR = Path(__file__).parent.resolve()


@pytest.mark.kubernetes
def test_genericjmx_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = DIR / "genericjmx-k8s.yaml"
    monitors = [
        {
            "type": "collectd/genericjmx",
            "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
            "serviceURL": "service:jmx:rmi:///jndi/rmi://{{.Host}}:{{.Port}}/jmxrmi",
            "username": "testuser",
            "password": "testing123",
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

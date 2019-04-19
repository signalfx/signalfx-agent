import pytest

from tests.helpers.kubernetes.utils import run_k8s_monitors_test
from tests.helpers.util import get_monitor_dims_from_selfdescribe, get_monitor_metrics_from_selfdescribe

pytestmark = [pytest.mark.kubernetes_volumes, pytest.mark.monitor_without_endpoints]


@pytest.mark.kubernetes
def test_kubernetes_volumes_in_k8s(agent_image, minikube, k8s_test_timeout, k8s_namespace):
    monitors = [{"type": "kubernetes-volumes", "kubeletAPI": {"skipVerify": True, "authType": "serviceAccount"}}]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        expected_metrics=get_monitor_metrics_from_selfdescribe(monitors[0]["type"]),
        expected_dims=get_monitor_dims_from_selfdescribe(monitors[0]["type"]),
        test_timeout=k8s_test_timeout,
    )

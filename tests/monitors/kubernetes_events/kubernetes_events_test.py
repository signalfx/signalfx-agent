from functools import partial as p

import pytest

from helpers.kubernetes.utils import run_k8s_with_agent
from helpers.util import ensure_always, wait_for

pytestmark = [pytest.mark.kubernetes_events, pytest.mark.monitor_without_endpoints]


def has_event(fake_services, event_dict):
    event_type = event_dict["reason"]
    kubernetes_kind = event_dict["involvedObjectKind"]
    for event in fake_services.events:
        if event.eventType == event_type:
            for dim in event.dimensions:
                if dim.key == "kubernetes_kind" and dim.value == kubernetes_kind:
                    return True
    return False


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_k8s_events_with_whitelist(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    expected_events = [
        {"reason": "Pulled", "involvedObjectKind": "Pod"},
        {"reason": "Created", "involvedObjectKind": "Pod"},
        {"reason": "Started", "involvedObjectKind": "Pod"},
    ]
    monitors = [
        {
            "type": "kubernetes-events",
            "kubernetesAPI": {"authType": "serviceAccount"},
            "whitelistedEvents": expected_events,
        }
    ]
    with run_k8s_with_agent(agent_image, minikube, monitors, observer=k8s_observer, namespace=k8s_namespace) as [
        backend,
        agent,
    ]:
        for expected_event in expected_events:
            assert wait_for(p(has_event, backend, expected_event), k8s_test_timeout), (
                "timed out waiting for event '%s'!\n\nAGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n"
                % (expected_event, agent.get_status(), agent.get_container_logs())
            )


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_k8s_events_without_whitelist(agent_image, minikube, k8s_observer, k8s_namespace):
    monitors = [{"type": "kubernetes-events", "kubernetesAPI": {"authType": "serviceAccount"}}]
    with run_k8s_with_agent(agent_image, minikube, monitors, observer=k8s_observer, namespace=k8s_namespace) as [
        backend,
        agent,
    ]:
        assert ensure_always(lambda: not backend.events, 30), (
            "event received!\n\nAGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n"
            % (agent.get_status(), agent.get_container_logs())
        )

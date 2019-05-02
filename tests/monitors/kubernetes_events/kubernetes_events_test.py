from functools import partial as p

import pytest

from tests.helpers.util import ensure_always, wait_for

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


@pytest.mark.kubernetes
def test_k8s_events_with_whitelist(agent_image, minikube, k8s_test_timeout, k8s_namespace):
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
    with minikube.run_agent(agent_image, monitors=monitors, namespace=k8s_namespace) as [_, fake_services]:
        for expected_event in expected_events:
            assert wait_for(p(has_event, fake_services, expected_event), k8s_test_timeout), (
                "timed out waiting for event '%s'!" % expected_event
            )


@pytest.mark.kubernetes
def test_k8s_events_without_whitelist(agent_image, minikube, k8s_namespace):
    monitors = [{"type": "kubernetes-events", "kubernetesAPI": {"authType": "serviceAccount"}}]
    with minikube.run_agent(agent_image, monitors=monitors, namespace=k8s_namespace) as [_, fake_services]:
        assert ensure_always(lambda: not fake_services.events, 30), "event received!"

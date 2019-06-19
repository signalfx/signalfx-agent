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
def test_k8s_events_with_whitelist(k8s_cluster):
    config = """
      monitors:
       - type: kubernetes-events
         whitelistedEvents:
          - reason: Pulled
            involvedObjectKind: Pod
          - reason: Created
            involvedObjectKind: Pod
          - reason: Started
            involvedObjectKind: Pod
      """
    with k8s_cluster.run_agent(config) as agent:
        for expected_event in [
            {"reason": "Pulled", "involvedObjectKind": "Pod"},
            {"reason": "Created", "involvedObjectKind": "Pod"},
            {"reason": "Started", "involvedObjectKind": "Pod"},
        ]:
            assert wait_for(p(has_event, agent.fake_services, expected_event)), (
                "timed out waiting for event '%s'!" % expected_event
            )


@pytest.mark.kubernetes
def test_k8s_events_without_whitelist(k8s_cluster):
    config = """
      monitors:
        - type: kubernetes-events
      """
    with k8s_cluster.run_agent(config) as agent:
        assert ensure_always(lambda: not agent.fake_services.events, 30), "event received!"

from tests.helpers import util
from tests.helpers.agent import Agent
from tests.helpers.util import wait_for


def verify(agent, metadata, included_only, timeout=util.DEFAULT_TIMEOUT):
    metrics = metadata.included_metrics if included_only else metadata.all_metrics
    wait_for(lambda: frozenset(agent.fake_services.datapoints_by_metric) == metrics, timeout_seconds=timeout)
    assert frozenset(agent.fake_services.datapoints_by_metric) == metrics


def verify_included_metrics(config, metadata, timeout=util.DEFAULT_TIMEOUT):
    with Agent.run(config) as agent:
        verify(agent, metadata, included_only=True, timeout=timeout)


def verify_all_metrics(config, metadata, timeout=util.DEFAULT_TIMEOUT):
    with Agent.run(config) as agent:
        verify(agent, metadata, included_only=False, timeout=timeout)

from tests.helpers import util
from tests.helpers.agent import Agent
from tests.helpers.util import wait_for


def verify(agent, metrics, timeout=util.DEFAULT_TIMEOUT):
    wait_for(lambda: frozenset(agent.fake_services.datapoints_by_metric) == metrics, timeout_seconds=timeout)
    assert frozenset(agent.fake_services.datapoints_by_metric) == metrics


def verify_included_metrics(config, metadata, timeout=util.DEFAULT_TIMEOUT):
    with Agent.run(config) as agent:
        verify(agent, metadata.included_metrics, timeout=timeout)


def verify_all_metrics(config, metadata, timeout=util.DEFAULT_TIMEOUT):
    with Agent.run(config) as agent:
        verify(agent, metadata.all_metrics, timeout=timeout)


def verify_custom(config, metrics, timeout=util.DEFAULT_TIMEOUT):
    with Agent.run(config) as agent:
        verify(agent, metrics, timeout=timeout)

from tests.helpers import util
from tests.helpers.agent import Agent
from tests.helpers.util import wait_for_assertion


def verify(agent, expected_metrics, timeout=util.DEFAULT_TIMEOUT):
    def test():
        assert frozenset(agent.fake_services.datapoints_by_metric) == expected_metrics

    wait_for_assertion(test, timeout_seconds=timeout)


def verify_expected_is_subset(agent, expected_metrics, timeout=util.DEFAULT_TIMEOUT):
    def test():
        assert expected_metrics <= frozenset(agent.fake_services.datapoints_by_metric)

    wait_for_assertion(test, timeout_seconds=timeout)


def verify_included_metrics(config, metadata, timeout=util.DEFAULT_TIMEOUT):
    with Agent.run(config) as agent:
        verify(agent, metadata.included_metrics, timeout=timeout)


def verify_all_metrics(config, metadata, timeout=util.DEFAULT_TIMEOUT):
    with Agent.run(config) as agent:
        verify(agent, metadata.all_metrics, timeout=timeout)


def verify_custom(config, metrics, timeout=util.DEFAULT_TIMEOUT):
    with Agent.run(config) as agent:
        verify(agent, metrics, timeout=timeout)

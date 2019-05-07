import pytest

from tests.helpers import util
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_log_message
from tests.helpers.util import wait_for_assertion, ensure_always


def verify(agent, expected_metrics, timeout=util.DEFAULT_TIMEOUT):
    def test():
        if has_log_message(agent.output.lower(), "error"):
            pytest.fail("error found in agent output!")

        assert frozenset(agent.fake_services.datapoints_by_metric) == expected_metrics

    if not expected_metrics:
        ensure_always(test, timeout_seconds=timeout)
    else:
        wait_for_assertion(test, timeout_seconds=timeout)


def verify_expected_is_subset(agent, expected_metrics, timeout=util.DEFAULT_TIMEOUT):
    def test():
        if has_log_message(agent.output.lower(), "error"):
            pytest.fail("error found in agent output!")

        assert expected_metrics <= frozenset(agent.fake_services.datapoints_by_metric)

    wait_for_assertion(test, timeout_seconds=timeout)


def verify_custom(config, metrics, timeout=util.DEFAULT_TIMEOUT):
    with Agent.run(config) as agent:
        verify(agent, metrics, timeout=timeout)


def verify_included_metrics(config, metadata, timeout=util.DEFAULT_TIMEOUT):
    with Agent.run(config) as agent:
        verify(agent, metadata.included_metrics, timeout=timeout)


def verify_all_metrics(config, metadata, timeout=util.DEFAULT_TIMEOUT):
    with Agent.run(config) as agent:
        verify(agent, metadata.all_metrics, timeout=timeout)

import pytest
from tests.helpers import util
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_log_message
from tests.helpers.util import wait_for_assertion


def verify(agent, expected_metrics, timeout=util.DEFAULT_TIMEOUT, check_errors=True):
    def test():
        if check_errors and has_log_message(agent.output.lower(), "error"):
            pytest.fail("error found in agent output!")

        assert frozenset(agent.fake_services.datapoints_by_metric) == expected_metrics

    wait_for_assertion(test, timeout_seconds=timeout)


def verify_expected_is_subset(agent, expected_metrics, timeout=util.DEFAULT_TIMEOUT, check_errors=True):
    def test():
        if check_errors and has_log_message(agent.output.lower(), "error"):
            pytest.fail("error found in agent output!")

        assert expected_metrics <= frozenset(agent.fake_services.datapoints_by_metric)

    wait_for_assertion(test, timeout_seconds=timeout)


def verify_expected_is_superset(agent, expected_metrics, timeout=util.DEFAULT_TIMEOUT):
    def test():
        assert frozenset(agent.fake_services.datapoints_by_metric) <= frozenset(expected_metrics)

    wait_for_assertion(test, timeout_seconds=timeout)


def verify_custom(config, metrics, timeout=util.DEFAULT_TIMEOUT, check_errors=True):
    with Agent.run(config) as agent:
        verify(agent, metrics, timeout=timeout, check_errors=check_errors)


def verify_included_metrics(config, metadata, timeout=util.DEFAULT_TIMEOUT, check_errors=True):
    with Agent.run(config) as agent:
        verify(agent, metadata.included_metrics, timeout=timeout, check_errors=check_errors)


def verify_all_metrics(config, metadata, timeout=util.DEFAULT_TIMEOUT, check_errors=True):
    with Agent.run(config) as agent:
        verify(agent, metadata.all_metrics, timeout=timeout, check_errors=check_errors)

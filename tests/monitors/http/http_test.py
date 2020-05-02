from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import (
    has_datapoint_with_dim_key,
    all_datapoints_have_dim_key,
    has_datapoint_with_metric_name,
    has_datapoint_with_dim,
)
from tests.helpers.metadata import Metadata
from tests.helpers.util import wait_for
from tests.helpers.verify import run_agent_verify_default_metrics, verify_expected_is_subset

pytestmark = [pytest.mark.windows, pytest.mark.filesystems, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("http")
# will redirect to https://www.google.com
URL_HTTPS = "http://gogole.com"
# will not be compatible with https
URL_HTTP = "http://neverssl.com"
# dimensions available on every metrics
DIMS_GLOBAL = ["url"]
# crapy but default metrics could not be reported (tls expiry)
METRIC_TO_DIM = {}
# https has only one metric but contains dimension "isValid"
METRIC_TLS = "http.cert_expiry"
METRIC_TO_DIM[METRIC_TLS] = "isValid"
METRIC_CODE = "http.status_code"
METRIC_TO_DIM[METRIC_CODE] = "matchCode"
METRIC_LENGTH = "http.content_length"
METRIC_TIME = "http.response_time"
METRIC_TO_DIM[METRIC_LENGTH] = "matchRegex"
# dimensions which could be unavailable depending on the configuration
DIMS_OPTIONAL = [METRIC_TO_DIM[METRIC_LENGTH], METRIC_TO_DIM[METRIC_TLS]]
# dimensions available on at least one metric
DIMS_ALWAYS = ["method", "desiredCode", "serverName", METRIC_TO_DIM[METRIC_CODE]] + DIMS_OPTIONAL
# for now only https metric could be not reported (when https unavailable)
METRICS_OPTIONAL = {METRIC_TLS}


def check_values(dps):
    for dp in dps:
        # correct default status code should be 200
        if dp.metric == METRIC_CODE:
            assert dp.value.intValue == 200
        # check good possible values
        if dp.metric == METRIC_LENGTH:
            assert dp.value.intValue > 0
        if dp.metric == METRIC_TIME or dp.metric == METRIC_TLS:
            assert dp.value.doubleValue > 0


def test_http_all_metrics():
    # Config to get every possible metrics
    agent_config = dedent(
        f"""
        monitors:
        - type: http
          urls:
            - {URL_HTTPS}
        """
    )
    # every metrics should be reported for https site
    run_agent_verify_default_metrics(agent_config, METADATA)


def test_http_minimal_metrics():
    # config to get only minimal metrics
    with Agent.run(
        f"""
        monitors:
        - type: http
          urls:
            - {URL_HTTP}
        """
    ) as agent:
        # https metric(s) should not be reported for http site
        verify_expected_is_subset(agent, METADATA.all_metrics - METRICS_OPTIONAL)


def test_http_all_stats():
    # Config to get every possible dimensions (and metrics so) to OK
    with Agent.run(
        f"""
        monitors:
        - type: http
          urls:
          - {URL_HTTPS}
          regex: ".*"
        """
    ) as agent:
        for dim in DIMS_GLOBAL:
            # global dimensions should be on every metrics
            assert wait_for(
                p(all_datapoints_have_dim_key, agent.fake_services, dim)
            ), "Didn't get http datapoints with {} global dimension".format(dim)
        for dim in DIMS_ALWAYS:
            # dimensions which should be available on one metric at least
            assert has_datapoint_with_dim_key(
                agent.fake_services, dim
            ), "Didn't get http datapoints with {} dimension".format(dim)
        # tls metric should be here
        assert has_datapoint_with_metric_name(
            agent.fake_services, METRIC_TLS
        ), "Didn't get http datapoints with metric name {}".format(METRIC_TLS)
        # tls should be valid
        assert has_datapoint_with_dim(
            agent.fake_services, METRIC_TO_DIM[METRIC_TLS], "true"
        ), "Didn't get http datapoints with valid tls"
        # regex should match
        assert has_datapoint_with_dim(
            agent.fake_services, METRIC_TO_DIM[METRIC_LENGTH], "true"
        ), "Didn't get http datapoints with regex wildcard match"
        # code should match
        assert has_datapoint_with_dim(
            agent.fake_services, METRIC_TO_DIM[METRIC_CODE], "true"
        ), "Didn't get http datapoints with valid status code"
        check_values(agent.fake_services.datapoints)


def test_http_minimal_stats():
    # config to get only minimal dimensions
    with Agent.run(
        f"""
        monitors:
        - type: http
          urls:
          - {URL_HTTP}
          desiredCode: 500
        """
    ) as agent:
        # optional dimensions should not be available
        for dim in DIMS_OPTIONAL:
            assert not wait_for(
                p(has_datapoint_with_dim_key, agent.fake_services, dim)
            ), "Got http datapoints with {} dimension and should not according to config".format(dim)
        # tls metric should not be available
        assert not has_datapoint_with_metric_name(
            agent.fake_services, METRIC_TLS
        ), "Got http datapoints with metric name {} whereas ocnfigured site is only http".format(METRIC_TLS)
        # the correct 200 code should not match 500 desired one
        assert has_datapoint_with_dim(
            agent.fake_services, METRIC_TO_DIM[METRIC_CODE], "false"
        ), "Didn't get http datapoints with valid status code"
        check_values(agent.fake_services.datapoints)


def test_http_noredirect():
    with Agent.run(
        f"""
        monitors:
        - type: http
          urls:
          - {URL_HTTPS}
          noRedirects: true
          desiredCode: 301
          regex: "$a"
        """
    ) as agent:
        # not a 200 code but should match the desired one
        assert wait_for(
            p(has_datapoint_with_dim, agent.fake_services, METRIC_TO_DIM[METRIC_CODE], "true")
        ), "Didn't get http datapoints with valid status code (should not be redirected)"
        # the regex should never match anything
        assert has_datapoint_with_dim(
            agent.fake_services, METRIC_TO_DIM[METRIC_LENGTH], "false"
        ), "Didn't get http datapoints with valid status code"
        for dp in agent.fake_services.datapoints:
            if dp.metric == METRIC_CODE:
                print(dp.dimensions)
                assert dp.value.intValue == 301

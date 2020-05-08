from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import all_datapoints_have_dim_key, has_datapoint_with_metric_name
from tests.helpers.metadata import Metadata
from tests.helpers.util import wait_for
from tests.helpers.verify import run_agent_verify_default_metrics, verify_expected_is_subset

pytestmark = [pytest.mark.http, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("http")
# will redirect to https://www.google.com
URL_HTTPS = "http://gogole.com"
URL_HTTPS_SELFSIGNED = "https://self-signed.badssl.com/"
URL_HTTPS_EXPIRED = "https://expired.badssl.com/"
URL_HTTPS_SELFSIGNED = "https://self-signed.badssl.com/"
# will not be compatible with https
URL_HTTP = "http://neverssl.com"
# dimensions available on every metrics
DIMS_GLOBAL = ["url"]
# crapy but default metrics could not be reported (tls expiry)
METRIC_CODE = "http.status_code"
METRIC_LENGTH = "http.content_length"
METRIC_TIME = "http.response_time"
METRIC_CERT_VALID = "http.cert_valid"
METRIC_CERT_EXPIRY = "http.cert_expiry"
METRIC_REGEX = "http.regex_matched"
METRIC_CODE_MATCH = "http.code_matched"
METRICS_OPTIONAL = {METRIC_CERT_VALID, METRIC_CERT_EXPIRY, METRIC_REGEX}


def check_values(dps, status_code, code=1, regex=1, cert=1):
    for dp in dps:
        # correct default status code should be 200
        if dp.metric == METRIC_CODE:
            assert status_code in (dp.value.intValue, 0)
        # check good possible values
        if dp.metric == METRIC_LENGTH:
            assert dp.value.intValue > 0
        if dp.metric == METRIC_CERT_VALID:
            assert dp.value.intValue == cert
        if dp.metric == METRIC_REGEX:
            assert dp.value.intValue == regex
        if dp.metric == METRIC_CODE_MATCH:
            assert dp.value.intValue > 0 or code
        if dp.metric == METRIC_TIME or dp.metric == METRIC_CERT_EXPIRY:
            assert dp.value.doubleValue > 0


def test_http_all_metrics():
    # Config to get every possible metrics
    agent_config = dedent(
        f"""
        monitors:
        - type: http
          urls:
            - {URL_HTTPS}
          regex: ".*"
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
    # Config to get every metrics to OK
    with Agent.run(
        f"""
        monitors:
        - type: http
          urls:
          - {URL_HTTPS}
          regex: ".*"
          desiredCode: 200 # default
        """
    ) as agent:
        for dim in DIMS_GLOBAL:
            # global dimensions should be on every metrics
            assert wait_for(
                p(all_datapoints_have_dim_key, agent.fake_services, dim)
            ), "Didn't get http datapoints with {} global dimension".format(dim)
        check_values(agent.fake_services.datapoints, agent.config["monitors"][0]["desiredCode"])


def test_http_minimal_stats():
    # config to get KO on regex, code and test no redirect
    with Agent.run(
        f"""
        monitors:
        - type: http
          urls:
          - {URL_HTTPS}
          noRedirects: true
          regex: "$a"
        """
    ) as agent:
        # tls metric should not be available
        for metric in METRICS_OPTIONAL:
            assert not has_datapoint_with_metric_name(
                agent.fake_services, metric
            ), "Got http datapoints with metric name {} but should not".format(metric)
        # 301 because not redirectd and regex should never match
        check_values(agent.fake_services.datapoints, 0, code=0, regex=0)


def test_http_tls_stats():
    # config to get bad tls
    with Agent.run(
        f"""
        monitors:
        - type: http
          urls:
          - {URL_HTTPS_SELFSIGNED}
          - {URL_HTTPS_EXPIRED}
        """
    ) as agent:
        # url should has a bad certificate
        check_values(agent.fake_services.datapoints, 200, cert=0)

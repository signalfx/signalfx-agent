import sys
from functools import partial as p
from textwrap import dedent

import pytest
import copy
from tests.helpers.agent import Agent
from tests.helpers.assertions import (
    has_datapoint, 
    has_no_datapoint, 
    has_datapoint_with_dim_key,
    all_datapoints_have_dim_key,
    has_datapoint_with_metric_name,
    has_datapoint_with_dim,
)
from tests.helpers.metadata import Metadata
from tests.helpers.util import ensure_always, wait_for
from tests.helpers.verify import run_agent_verify_default_metrics, verify_expected_is_subset

pytestmark = [pytest.mark.windows, pytest.mark.filesystems, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("http")
# will redirect to https://www.google.com
https_url = "http://gogole.com"
# will not be compatible with https
http_url = "http://neverssl.com"
# dimensions available on every metrics
global_dims = ["url"]
# crapy but default metrics could not be reported (tls expiry)
metrics_to_dimension = {}
# https has only one metric but contains dimension "isValid"
https_metric = "http.cert_expiry"
metrics_to_dimension[https_metric] = "isValid"
code_metric = "http.status_code"
metrics_to_dimension[code_metric] = "matchCode"
length_metric = "http.content_length"
time_metric = "http.response_time"
metrics_to_dimension[length_metric] = "matchRegex"
# dimensions which could be unavailable depending on the configuration
optional_dims = [metrics_to_dimension[length_metric], metrics_to_dimension[https_metric]]
# dimensions available on at least one metric
always_dims = ["method", "desiredCode", "serverName", metrics_to_dimension[code_metric]] + optional_dims
# for now only https metric could be not reported (when https unavailable)
optional_metrics = {https_metric}

def check_values(dps):
    for dp in dps:
        # correct default status code should be 200
        if dp.metric == code_metric:
            assert dp.value.intValue == 200
        # check good possible values
        if dp.metric == length_metric:
            assert dp.value.intValue > 0
        if dp.metric == time_metric or dp.metric == https_metric:
            assert dp.value.doubleValue > 0

def test_http_all_metrics():
    # Config to get every possible metrics
    agent_config = dedent(
        f"""
        monitors:
        - type: http
          urls:
            - {https_url}
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
            - {http_url}
        """
    ) as agent:
        # https metric(s) should not be reported for http site
        verify_expected_is_subset(agent, METADATA.all_metrics - optional_metrics)

def test_http_all_stats():
    # Config to get every possible dimensions (and metrics so) to OK
    with Agent.run(
        f"""
        monitors:
        - type: http
          urls:
          - {https_url}
          regex: ".*"
        """
    ) as agent:
        for dim in global_dims:
            # global dimensions should be on every metrics
            assert wait_for(p(all_datapoints_have_dim_key, agent.fake_services, dim)), (
                "Didn't get http datapoints with {} global dimension".format(dim)
            )
        for dim in always_dims:
            # dimensions which should be available on one metric at least 
            assert has_datapoint_with_dim_key(agent.fake_services, dim), (
                "Didn't get http datapoints with {} dimension".format(dim)
            )
        # tls metric should be here
        assert has_datapoint_with_metric_name(agent.fake_services, https_metric), (
            "Didn't get http datapoints with metric name {}".format(https_metric)
        )
        # tls should be valid
        assert has_datapoint_with_dim(agent.fake_services, metrics_to_dimension[https_metric], "true"), (
            "Didn't get http datapoints with valid tls"
        )
        # regex should match
        assert has_datapoint_with_dim(agent.fake_services, metrics_to_dimension[length_metric], "true"), (
            "Didn't get http datapoints with regex wildcard match"
        )
        # code should match
        assert has_datapoint_with_dim(agent.fake_services, metrics_to_dimension[code_metric], "true"), (
            "Didn't get http datapoints with valid status code"
        )
        check_values(agent.fake_services.datapoints)

def test_http_minimal_stats():
    # config to get only minimal dimensions
    with Agent.run(
        f"""
        monitors:
        - type: http
          urls:
          - {http_url}
          desiredCode: 500
        """
    ) as agent:
        # optional dimensions should not be available
        for dim in optional_dims:
            assert not wait_for(p(has_datapoint_with_dim_key, agent.fake_services, dim)), (
                "Got http datapoints with {} dimension and should not according to config".format(dim)
            )
        # tls metric should not be available
        assert not has_datapoint_with_metric_name(agent.fake_services, https_metric), (
            "Got http datapoints with metric name {} whereas ocnfigured site is only http".format(https_metric)
        )
        # the correct 200 code should not match 500 desired one
        assert has_datapoint_with_dim(agent.fake_services, metrics_to_dimension[code_metric], "false"), (
            "Didn't get http datapoints with valid status code"
        )
        check_values(agent.fake_services.datapoints)

def test_http_noredirect():
    with Agent.run(
        f"""
        monitors:
        - type: http
          urls:
          - {https_url}
          noRedirects: true
          desiredCode: 301
          regex: "$a"
        """
    ) as agent:
        # not a 200 code but should match the desired one
        assert wait_for(p(has_datapoint_with_dim, agent.fake_services, metrics_to_dimension[code_metric], "true")), (
            "Didn't get http datapoints with valid status code (should not be redirected)"
        )
        # the regex should never match anything
        assert has_datapoint_with_dim(agent.fake_services, metrics_to_dimension[length_metric], "false"), (
            "Didn't get http datapoints with valid status code"
        )
        for dp in agent.fake_services.datapoints:
            if dp.metric == code_metric:
                print (dp.dimensions)
                assert dp.value.intValue == 301

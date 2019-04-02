"""
Tests for the conviva monitor
"""
import json
import os
import pytest
import re
import requests
import time
from functools import partial as p
from textwrap import dedent

from tests.helpers.assertions import *
from tests.helpers.util import run_agent, wait_for, ensure_always, get_agent_status

pytestmark = [pytest.mark.conviva, pytest.mark.monitor_without_endpoints]

CONVIVA_PULSE_API_URL = os.environ.get("CONVIVA_PULSE_API_URL", "https://api.conviva.com/insights/2.4/")
CONVIVA_PULSE_USERNAME = os.environ.get("CONVIVA_PULSE_USERNAME")
CONVIVA_PULSE_PASSWORD = os.environ.get("CONVIVA_PULSE_PASSWORD")

pytest.skip("Temporarily disable test while broken", allow_module_level=True)

if not CONVIVA_PULSE_USERNAME or not CONVIVA_PULSE_PASSWORD:
    pytest.skip("CONVIVA_PULSE_USERNAME and/or CONVIVA_PULSE_PASSWORD env vars not set", allow_module_level=True)


def get_conviva_json(path, max_attempts=3):
    url = CONVIVA_PULSE_API_URL.rstrip("/") + "/" + path.lstrip("/")
    auth = requests.auth.HTTPBasicAuth(CONVIVA_PULSE_USERNAME, CONVIVA_PULSE_PASSWORD)
    json_resp = None
    attempts = 0
    while attempts < max_attempts:
        with requests.get(url, auth=auth) as resp:
            json_resp = json.loads(resp.text)
            if resp.status_code == 200:
                break
        time.sleep(5)
        attempts += 1
    return json_resp


@pytest.fixture(scope="module")
def conviva_accounts():
    accounts_json = get_conviva_json("accounts.json")
    assert (
        "accounts" in accounts_json.keys() and len(accounts_json["accounts"].keys()) > 0
    ), "No accounts found in accounts.json response:\n%s" % str(accounts_json)
    return list(accounts_json["accounts"].keys())


@pytest.fixture(scope="module")
def conviva_filters():
    filters_json = get_conviva_json("filters.json")
    assert filters_json, "No filters found in filters.json response:\n%s" % str(filters_json)
    return list(filters_json.values())


@pytest.fixture(scope="module")
def conviva_metriclens_dimensions():
    metriclens_dimensions_json = get_conviva_json("metriclens_dimension_list.json")
    assert metriclens_dimensions_json, (
        "No metriclens dimensions found in metriclens_dimension_list.json response:\n%s" % metriclens_dimensions_json
    )
    return list(metriclens_dimensions_json.keys())


def get_dim_key(metriclens_dimension):
    return re.sub(r"\W", "_", metriclens_dimension)


def test_conviva_basic():
    with run_agent(
        dedent(
            f"""
        monitors:
        - type: conviva
          pulseUsername: {{"#from": "env:CONVIVA_PULSE_USERNAME"}}
          pulsePassword: {{"#from": "env:CONVIVA_PULSE_PASSWORD"}}
    """
        ),
        debug=False,
    ) as [backend, get_output, agent_config]:
        assert wait_for(lambda: len(backend.datapoints) > 0), "Didn't get conviva datapoints"
        pattern = re.compile("^conviva\.quality_metriclens\..*")
        assert ensure_always(
            p(all_datapoints_have_metric_name_and_dims, backend, pattern, {"filter": "All Traffic"})
        ), "Received datapoints without metric quality_metriclens or {filter: All Traffic} dimension"
        config_path = agent_config(None)
        agent_status = get_agent_status(config_path)
        assert CONVIVA_PULSE_PASSWORD not in agent_status, (
            "cleartext password(s) found in agent status output!\n\n%s\n" % agent_status
        )
        agent_output = get_output()
        assert CONVIVA_PULSE_PASSWORD not in agent_output, (
            "cleartext password(s) found in agent output!\n\n%s\n" % agent_output
        )


def test_conviva_extra_dimensions():
    with run_agent(
        dedent(
            f"""
        monitors:
        - type: conviva
          pulseUsername: {{"#from": "env:CONVIVA_PULSE_USERNAME"}}
          pulsePassword: {{"#from": "env:CONVIVA_PULSE_PASSWORD"}}
          extraDimensions:
            metric_source: conviva
            mydim: foo
    """
        ),
        debug=False,
    ) as [backend, _, _]:
        assert wait_for(lambda: len(backend.datapoints) > 0), "Didn't get conviva datapoints"
        assert ensure_always(
            p(all_datapoints_have_dims, backend, {"metric_source": "conviva", "mydim": "foo"})
        ), "Received conviva datapoints without extra dimensions"


def test_conviva_single_metric():
    with run_agent(
        dedent(
            f"""
        monitors:
        - type: conviva
          pulseUsername: {{"#from": "env:CONVIVA_PULSE_USERNAME"}}
          pulsePassword: {{"#from": "env:CONVIVA_PULSE_PASSWORD"}}
          metricConfigs:
          - metricParameter: concurrent_plays
    """
        ),
        debug=False,
    ) as [backend, _, _]:
        assert wait_for(lambda: len(backend.datapoints) > 0), "Didn't get conviva datapoints"
        assert ensure_always(
            p(all_datapoints_have_metric_name, backend, "conviva.concurrent_plays")
        ), "Received conviva datapoints for other metrics"


def test_conviva_multi_metric():
    with run_agent(
        dedent(
            f"""
        monitors:
        - type: conviva
          pulseUsername: {{"#from": "env:CONVIVA_PULSE_USERNAME"}}
          pulsePassword: {{"#from": "env:CONVIVA_PULSE_PASSWORD"}}
          metricConfigs:
          - metricParameter: concurrent_plays
          - metricParameter: plays
    """
        ),
        debug=False,
    ) as [backend, _, _]:
        assert wait_for(
            p(has_datapoint_with_metric_name, backend, "conviva.concurrent_plays")
        ), "Didn't get conviva datapoints for metric concurrent_plays"
        assert wait_for(
            p(has_datapoint_with_metric_name, backend, "conviva.plays")
        ), "Didn't get conviva datapoints for metric plays"


def test_conviva_metriclens():
    with run_agent(
        dedent(
            f"""
        monitors:
        - type: conviva
          pulseUsername: {{"#from": "env:CONVIVA_PULSE_USERNAME"}}
          pulsePassword: {{"#from": "env:CONVIVA_PULSE_PASSWORD"}}
          metricConfigs:
          - metricParameter: audience_metriclens
          - metricParameter: quality_metriclens
    """
        ),
        debug=False,
    ) as [backend, _, _]:
        pattern = re.compile("^conviva\.audience_metriclens\..*")
        assert wait_for(
            p(has_datapoint_with_metric_name, backend, pattern)
        ), "Didn't get conviva datapoints for metriclens audience_metriclens"
        pattern = re.compile("^conviva\.quality_metriclens\..*")
        assert wait_for(
            p(has_datapoint_with_metric_name, backend, pattern)
        ), "Didn't get conviva datapoints for metriclens quality_metriclens"


def test_conviva_single_metriclens_dimension(conviva_metriclens_dimensions):
    with run_agent(
        dedent(
            f"""
        monitors:
        - type: conviva
          pulseUsername: {{"#from": "env:CONVIVA_PULSE_USERNAME"}}
          pulsePassword: {{"#from": "env:CONVIVA_PULSE_PASSWORD"}}
          metricConfigs:
          - metricParameter: quality_metriclens
            metricLensDimensions:
            - {conviva_metriclens_dimensions[0]}
    """
        ),
        debug=False,
    ) as [backend, _, _]:
        assert wait_for(lambda: len(backend.datapoints) > 0), "Didn't get conviva datapoints"
        pattern = re.compile("^conviva\.quality_metriclens\..*")
        assert ensure_always(
            p(all_datapoints_have_metric_name, backend, pattern)
        ), "Received conviva datapoints for other metrics"
        assert ensure_always(p(all_datapoints_have_dim_key, backend, get_dim_key(conviva_metriclens_dimensions[0]))), (
            "Received conviva datapoints without %s dimension" % conviva_metriclens_dimensions[0]
        )


def test_conviva_multi_metriclens_dimension(conviva_metriclens_dimensions):
    with run_agent(
        dedent(
            f"""
        monitors:
        - type: conviva
          pulseUsername: {{"#from": "env:CONVIVA_PULSE_USERNAME"}}
          pulsePassword: {{"#from": "env:CONVIVA_PULSE_PASSWORD"}}
          metricConfigs:
          - metricParameter: quality_metriclens
            metricLensDimensions: {conviva_metriclens_dimensions}
    """
        ),
        debug=False,
    ) as [backend, _, _]:
        for dim in conviva_metriclens_dimensions:
            if dim != "CDNs":
                assert wait_for(p(has_datapoint_with_dim_key, backend, get_dim_key(dim))), (
                    "Didn't get conviva datapoints with %s dimension" % dim
                )


def test_conviva_all_metriclens_dimension(conviva_metriclens_dimensions):
    with run_agent(
        dedent(
            f"""
        monitors:
        - type: conviva
          pulseUsername: {{"#from": "env:CONVIVA_PULSE_USERNAME"}}
          pulsePassword: {{"#from": "env:CONVIVA_PULSE_PASSWORD"}}
          metricConfigs:
          - metricParameter: quality_metriclens
            metricLensDimensions:
            - _ALL_
    """
        ),
        debug=False,
    ) as [backend, _, _]:
        for dim in conviva_metriclens_dimensions:
            if dim != "CDNs":
                assert wait_for(p(has_datapoint_with_dim_key, backend, get_dim_key(dim))), (
                    "Didn't get conviva datapoints with %s dimension" % dim
                )


def test_conviva_exclude_metriclens_dimension(conviva_metriclens_dimensions):
    with run_agent(
        dedent(
            f"""
        monitors:
        - type: conviva
          pulseUsername: {{"#from": "env:CONVIVA_PULSE_USERNAME"}}
          pulsePassword: {{"#from": "env:CONVIVA_PULSE_PASSWORD"}}
          metricConfigs:
          - metricParameter: quality_metriclens
            metricLensDimensions:
            - _ALL_
            excludeMetricLensDimensions:
            - CDNs
    """
        ),
        debug=False,
    ) as [backend, _, _]:
        assert wait_for(lambda: len(backend.datapoints) > 0), "Didn't get conviva datapoints"
        assert ensure_always(
            lambda: not has_datapoint_with_dim_key(backend, "CDNs")
        ), "Received datapoint with excluded CDNs dimension"


def test_conviva_metric_account(conviva_accounts):
    with run_agent(
        dedent(
            f"""
        monitors:
        - type: conviva
          pulseUsername: {{"#from": "env:CONVIVA_PULSE_USERNAME"}}
          pulsePassword: {{"#from": "env:CONVIVA_PULSE_PASSWORD"}}
          metricConfigs:
          - metricParameter: concurrent_plays
            account: {conviva_accounts[0]}
    """
        ),
        debug=False,
    ) as [backend, _, _]:
        assert wait_for(lambda: len(backend.datapoints) > 0), "Didn't get conviva datapoints"
        assert ensure_always(
            p(
                all_datapoints_have_metric_name_and_dims,
                backend,
                "conviva.concurrent_plays",
                {"account": conviva_accounts[0]},
            )
        ), (
            "Received conviva datapoints without metric conviva.concurrent_plays or {account: %s} dimension"
            % conviva_accounts[0]
        )


def test_conviva_single_filter(conviva_filters):
    with run_agent(
        dedent(
            f"""
        monitors:
        - type: conviva
          pulseUsername: {{"#from": "env:CONVIVA_PULSE_USERNAME"}}
          pulsePassword: {{"#from": "env:CONVIVA_PULSE_PASSWORD"}}
          metricConfigs:
          - metricParameter: concurrent_plays
            filters:
              - {conviva_filters[0]}
    """
        ),
        debug=False,
    ) as [backend, _, _]:
        assert wait_for(lambda: len(backend.datapoints) > 0), "Didn't get conviva datapoints"
        assert ensure_always(
            p(
                all_datapoints_have_metric_name_and_dims,
                backend,
                "conviva.concurrent_plays",
                {"filter": conviva_filters[0]},
            )
        ), (
            "Received conviva datapoints without metric conviva.concurrent_plays or {filter: %s} dimension"
            % conviva_filters[0]
        )


def test_conviva_multi_filter(conviva_filters):
    with run_agent(
        dedent(
            f"""
        monitors:
        - type: conviva
          pulseUsername: {{"#from": "env:CONVIVA_PULSE_USERNAME"}}
          pulsePassword: {{"#from": "env:CONVIVA_PULSE_PASSWORD"}}
          metricConfigs:
          - metricParameter: concurrent_plays
            filters: {conviva_filters[:3]}
    """
        ),
        debug=False,
    ) as [backend, _, _]:
        for cf in conviva_filters[:3]:
            assert wait_for(p(has_datapoint, backend, "conviva.concurrent_plays", {"filter": cf})), (
                "Didn't get conviva datapoints for metric concurrent_plays with dimension {filter: %s}" % cf
            )


def test_conviva_all_filter(conviva_filters):
    with run_agent(
        dedent(
            f"""
        monitors:
        - type: conviva
          pulseUsername: {{"#from": "env:CONVIVA_PULSE_USERNAME"}}
          pulsePassword: {{"#from": "env:CONVIVA_PULSE_PASSWORD"}}
          metricConfigs:
          - metricParameter: concurrent_plays
            maxFiltersPerRequest: 99
            filters:
              - _ALL_
    """
        ),
        debug=False,
    ) as [backend, _, _]:
        for cf in conviva_filters:
            assert wait_for(p(has_datapoint, backend, "conviva.concurrent_plays", {"filter": cf})), (
                "Didn't get conviva datapoints for metric concurrent_plays with dimension {filter: %s}" % cf
            )

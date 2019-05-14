import time
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, has_datapoint_with_metric_name
from tests.helpers.metadata import Metadata
from tests.helpers.util import ensure_always, run_service, wait_for
from tests.helpers.verify import run_agent_verify_included_metrics, verify_expected_is_subset, run_agent_verify_all_metrics

pytestmark = [pytest.mark.docker_container_stats, pytest.mark.monitor_without_endpoints]


METADATA = Metadata.from_package("collectd/docker")


def test_docker_included():
    with run_service(
        "elasticsearch/6.6.1"
    ):  # just get a container that does some block io running so we have some stats
        run_agent_verify_included_metrics(
            f"""
            monitors:
            - type: collectd/docker
              dockerURL: unix:///var/run/docker.sock
            """,
            METADATA,
        )

ENHANCED_METRICS = METADATA.all_metrics

def test_docker_enhanced():
    with run_service(
        "elasticsearch/6.6.1"
    ):  # just get a container that does some block io running so we have some stats
        run_agent_verify_all_metrics(
            f"""
            monitors:
            - type: collectd/docker
              dockerURL: unix:///var/run/docker.sock
              collectNetworkStats: true
            """, METADATA)

import sys
from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, has_no_datapoint
from tests.helpers.metadata import Metadata
from tests.helpers.util import ensure_always, wait_for
from tests.helpers.verify import run_agent_verify, run_agent_verify_default_metrics

pytestmark = [pytest.mark.windows, pytest.mark.filesystems, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("filesystems")


def test_filesystems_default_metrics():
    agent_config = dedent(
        """
        monitors:
        - type: filesystems
        """
    )
    run_agent_verify_default_metrics(agent_config, METADATA)


@pytest.mark.skipif(sys.platform.startswith("win"), reason="does not run on windows")
def test_filesystems_default_excludes_logical_filesystems():
    with Agent.run(
        """
        monitors:
        - type: filesystems
        """
    ) as agent:
        assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"mountpoint": "/"}))
        assert ensure_always(
            p(has_no_datapoint, agent.fake_services, dimensions={"mountpoint": "/dev"}), timeout_seconds=5
        )
        agent.update_config(
            """
        monitors:
        - type: filesystems
          mountPoints: ["/", "/dev"]
        """
        )
        assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"mountpoint": "/dev"}))


def test_filesystems_mountpoint_filter():
    root_mount = "C:" if sys.platform.startswith("win") else "/"
    with Agent.run(
        f"""
        monitors:
        - type: filesystems
          mountPoints:
          - "{root_mount}"
        """
    ) as agent:
        assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"mountpoint": root_mount}))
        for dp in agent.fake_services.datapoints:
            for dim in dp.dimensions:
                if dim.key == "mountpoint":
                    assert dim.value == root_mount


@pytest.mark.skipif(sys.platform.startswith("win"), reason="does not run on windows")
def test_filesystems_fstype_filter():
    with Agent.run(
        """
        monitors:
        - type: filesystems
          fsTypes:
          - proc
        """
    ) as agent:
        assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"fs_type": "proc"}))
        assert ensure_always(
            p(has_no_datapoint, agent.fake_services, dimensions={"mountpoint": "/"}), timeout_seconds=5
        )


def test_filesystems_percentage_group():
    expected_metrics = METADATA.default_metrics | METADATA.metrics_by_group["percentage"]
    agent_config = dedent(
        """
        monitors:
        - type: filesystems
          extraGroups: [percentage]
        """
    )
    run_agent_verify(agent_config, expected_metrics)


@pytest.mark.skipif(sys.platform.startswith("win"), reason="does not run on windows")
def test_filesystems_inodes_flag():
    expected_metrics = METADATA.default_metrics | METADATA.metrics_by_group["inodes"]

    agent_config = dedent(
        """
        monitors:
        - type: filesystems
          extraGroups: [inodes]
        """
    )
    run_agent_verify(agent_config, expected_metrics)


def test_filesystems_all_metrics():
    expected_metrics = METADATA.all_metrics
    if sys.platform.startswith("win"):
        expected_metrics = expected_metrics - METADATA.metrics_by_group["inodes"]

    agent_config = dedent(
        """
        monitors:
        - type: filesystems
          extraMetrics: ["*"]
        """
    )
    run_agent_verify(agent_config, expected_metrics)

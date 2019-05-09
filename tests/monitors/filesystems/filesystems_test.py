import sys
from textwrap import dedent

import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify_custom, verify_included_metrics

pytestmark = [pytest.mark.windows, pytest.mark.filesystems, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("filesystems")


def test_filesystems_included_metrics():
    agent_config = dedent(
        """
        monitors:
        - type: filesystems
        """
    )
    verify_included_metrics(agent_config, METADATA)


def test_filesystems_mountpoint_filter():
    expected_metrics = frozenset(["disk.summary_utilization"])
    agent_config = dedent(
        """
        procPath: /proc
        monitors:
        - type: filesystems
          mountPoints:
          - "!*"
        """
    )
    verify_custom(agent_config, expected_metrics)


def test_filesystems_fstype_filter():
    expected_metrics = frozenset(["disk.summary_utilization"])
    agent_config = dedent(
        """
        procPath: /proc
        monitors:
        - type: filesystems
          fsTypes:
          - "!*"
        """
    )
    verify_custom(agent_config, expected_metrics)


def test_filesystems_logical_flag():
    expected_metrics = METADATA.included_metrics | METADATA.metrics_by_group["logical"]
    agent_config = dedent(
        """
        procPath: /proc
        monitors:
        - type: filesystems
          includeLogical: true
        """
    )
    verify_custom(agent_config, expected_metrics)


def test_filesystems_inodes_flag():
    expected_metrics = METADATA.included_metrics
    if sys.platform == "linux":
        expected_metrics = expected_metrics | METADATA.metrics_by_group["inodes"]
    agent_config = dedent(
        """
        procPath: /proc
        monitors:
        - type: filesystems
          reportInodes: true
        """
    )
    verify_custom(agent_config, expected_metrics)


def test_filesystems_extra_metrics():
    percent_inodes_used, df_inodes_used = "percent_inodes.used", "df_inodes.used"
    expected_metrics = METADATA.included_metrics | {percent_inodes_used, df_inodes_used}
    agent_config = dedent(
        f"""
        procPath: /proc
        monitors:
        - type: filesystems
          extraMetrics:
          - {percent_inodes_used}
          - {df_inodes_used}
        """
    )
    verify_custom(agent_config, expected_metrics)


def test_filesystems_all_metrics():
    expected_metrics = METADATA.included_metrics | METADATA.metrics_by_group["logical"]
    if sys.platform == "linux":
        expected_metrics = METADATA.all_metrics
    agent_config = dedent(
        """
        procPath: /proc
        monitors:
        - type: filesystems
          includeLogical: true
          reportInodes: true
        """
    )
    verify_custom(agent_config, expected_metrics)

import sys

import pytest

from tests.helpers.assertions import has_log_message
from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify_default_metrics, run_agent_verify_all_metrics

pytestmark = [
    pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows"),
    pytest.mark.windows_only,
    pytest.mark.windowslegacy,
]

METADATA = Metadata.from_package("windowslegacy")


def test_windowslegacy_default():
    agent = run_agent_verify_default_metrics(
        """
        monitors:
        - type: windows-legacy
        """,
        METADATA,
    )
    assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"


def test_windowslegacy_all():
    agent = run_agent_verify_all_metrics(
        """
        monitors:
        - type: windows-legacy
          extraMetrics: ["*"]
        """,
        METADATA,
    )
    assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

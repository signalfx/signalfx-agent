import sys

import pytest

from tests.helpers.assertions import http_status
from tests.helpers.metadata import Metadata
from tests.helpers.verify import run_agent_verify_included_metrics, run_agent_verify_all_metrics

pytestmark = [
    pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows"),
    pytest.mark.windows_only,
    pytest.mark.windowsiis,
]

METADATA = Metadata.from_package("windowsiis")


def test_windowsiis_included():
    run_agent_verify_included_metrics(
        """
        monitors:
        - type: windows-iis
        """,
        METADATA,
    )


def test_windowsiis_all():
    # Required to make sure a worker (w3wp process) has started for process metrics.
    assert http_status("http://localhost", [200]), "IIS should be running on localhost"

    run_agent_verify_all_metrics(
        """
        monitors:
        - type: windows-iis
          extraMetrics: ["*"]
        """,
        METADATA,
    )

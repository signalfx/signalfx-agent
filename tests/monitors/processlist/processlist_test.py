from functools import partial as p
from textwrap import dedent
import pytest
import sys

from tests.helpers.util import wait_for, run_agent
from tests.helpers.assertions import *

pytestmark = [
    pytest.mark.skipif(sys.platform != 'win32', reason="only runs on windows"),
    pytest.mark.windows,
    pytest.mark.processlist
]


def test_processlist():
    config = dedent("""
        monitors:
         - type: processlist
        """)
    with run_agent(config) as [backend, _, _]:
        assert wait_for(p(has_event_type, backend, "objects.top-info")), "Didn't get processlist events"

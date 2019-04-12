import sys
from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_event_type, has_log_message
from tests.helpers.util import wait_for

pytestmark = [
    pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows"),
    pytest.mark.windows_only,
    pytest.mark.processlist,
]


def test_processlist():
    config = dedent(
        """
        monitors:
         - type: processlist
        """
    )
    with Agent.run(config) as agent:
        assert wait_for(p(has_event_type, agent.fake_services, "objects.top-info")), "Didn't get processlist events"
        assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"

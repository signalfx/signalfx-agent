from tests.helpers.agent import Agent
from tests.helpers.assertions import has_log_message
from tests.helpers.util import wait_for

CONFIG = """
monitors:
  - type: collectd/rabbitmq
    host: 127.0.0.1
"""


def test_validation_required_log_output():
    with Agent.run(CONFIG) as agent:
        assert wait_for(
            lambda: has_log_message(
                agent.output, "error", "Validation error in field 'Config.port': port is a required field (got '0')"
            )
        ), "Didn't get validation error message"

from helpers.assertions import has_log_message
from helpers.util import run_agent, wait_for

CONFIG = """
monitors:
  - type: collectd/rabbitmq
    host: 127.0.0.1
"""


def test_validation_required_log_output():
    with run_agent(CONFIG) as [_, get_output, _]:
        assert wait_for(
            lambda: has_log_message(get_output(), "error", "Validation error in field 'port': required")
        ), "Didn't get validation error message"

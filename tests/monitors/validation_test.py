from functools import partial as p
import os
import string

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, run_agent, run_container
from tests.helpers.assertions import *

config = """
monitors:
  - type: collectd/rabbitmq
    host: 127.0.0.1
"""

def test_validation_required_log_output():
    with run_agent(config) as [backend, get_output, _]:
        assert wait_for(lambda: has_log_message(get_output(), "error", "Validation error in field 'port': required")), "Didn't get validation error message"

from functools import partial as p
import os
import string

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, run_agent, run_container
from tests.helpers.assertions import *

rabbitmq_config = string.Template("""
monitors:
  - type: collectd/rabbitmq
    host: $host
    port: 15672
    username: guest
    password: guest
    collectNodes: true
    collectChannels: true
""")

def test_rabbitmq():
    with run_container("rabbitmq:3.6-management") as rabbitmq_cont:
        config = rabbitmq_config.substitute(host=rabbitmq_cont.attrs["NetworkSettings"]["IPAddress"])

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "rabbitmq")), "Didn't get rabbitmq datapoints"


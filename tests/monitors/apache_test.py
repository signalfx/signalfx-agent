from functools import partial as p
import os
import string

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, run_agent, run_service, container_ip
from tests.helpers.assertions import *

apache_config = string.Template("""
monitors:
  - type: collectd/apache
    host: $host
    port: 80
""")

def test_apache():
    with run_service("apache") as apache_container:
        host = container_ip(apache_container)
        config = apache_config.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 80), 60), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "apache")), "Didn't get apache datapoints"


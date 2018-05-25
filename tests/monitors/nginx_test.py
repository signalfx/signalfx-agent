from functools import partial as p
import os
import string

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, run_agent, run_service
from tests.helpers.assertions import *

nginx_config = string.Template("""
monitors:
  - type: collectd/nginx
    host: $host
    port: 80
""")

def test_nginx():
    with run_service("nginx") as nginx_container:
        host = nginx_container.attrs["NetworkSettings"]["IPAddress"]
        config = nginx_config.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 80), 60), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "nginx")), "Didn't get nginx datapoints"


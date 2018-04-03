from functools import partial as p
import os
import string
import time

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, ensure_always, run_agent, run_service
from tests.helpers.assertions import *

config = """
observers:
  - type: docker
    pollIntervalSeconds: 2
monitors:
  - type: collectd/nginx
    discoveryRule: container_name =~ "nginx-discovery" && port == 80
"""

def test_basic_service_discovery():
    with run_agent(config) as [backend, get_output, _]:
        with run_service("nginx", name="nginx-discovery") as nginx_container:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "nginx")), "Didn't get nginx datapoints"
        # Let nginx be removed by docker observer and collectd restart
        time.sleep(5)
        backend.datapoints.clear()
        assert ensure_always(lambda: not has_datapoint_with_dim(backend, "plugin", "nginx"), 10)
        assert not has_log_message(get_output(), "error")

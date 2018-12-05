"""
Integration tests for the python runner using the collectd adapter.
"""
import os
import re
import signal
import string
import time
from functools import partial as p

import pytest
import redis

from helpers.assertions import has_datapoint_with_dim, regex_search_matches_output, tcp_socket_open
from helpers.util import BUNDLE_DIR, container_ip, run_agent, run_container, wait_for

pytestmark = [pytest.mark.pyrunner]

PID_RE = re.compile(r"runnerPID=(\d+)")

MONITOR_CONFIG = string.Template(
    """
monitors:
  - type: collectd/python
    moduleName: redis_info
    modulePaths:
     - "${bundle_root}/plugins/collectd/redis"
    pluginConfig:
      Host: $host
      Port: 6379
      Verbose: true
      Redis_uptime_in_seconds: "gauge"
"""
)


def test_python_runner_with_redis():
    with run_container("redis:4-alpine") as test_container:
        host = container_ip(test_container)
        config = MONITOR_CONFIG.substitute(host=host, bundle_root=BUNDLE_DIR)
        assert wait_for(p(tcp_socket_open, host, 6379), 60), "redis is not listening on port"

        redis_client = redis.StrictRedis(host=host, port=6379, db=0)
        assert wait_for(redis_client.ping, 60), "service didn't start"

        with run_agent(config) as [backend, get_output, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "redis_info")), "didn't get datapoints"

            assert wait_for(p(regex_search_matches_output, get_output, PID_RE.search))
            pid = int(PID_RE.search(get_output()).groups()[0])

            os.kill(pid, signal.SIGTERM)

            time.sleep(3)
            backend.datapoints.clear()

            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "redis_info")
            ), "didn't get datapoints after Python process was killed"

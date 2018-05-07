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
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.util import container_ip, run_agent, run_container, wait_for

pytestmark = [pytest.mark.pyrunner]

pidRE = re.compile(r'runnerPID=(\d+)')

monitor_config = string.Template("""
monitors:
  - type: collectd/python
    moduleName: redis_info
    modulePaths:
     - "/bundle/plugins/collectd/redis"
    pluginConfig:
      Host: $host
      Port: 6379
      Verbose: true
      Redis_uptime_in_seconds: "gauge"
""")


def test_python_runner_with_redis():
    with run_container('redis:4-alpine') as test_container:
        host = container_ip(test_container)
        config = monitor_config.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 6379), 60), "redis is not listening on port"

        redis_client = redis.StrictRedis(host=host, port=6379, db=0)
        assert wait_for(redis_client.ping, 60), "service didn't start"

        with run_agent(config) as [backend, get_output, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "redis_info")), \
                "didn't get datapoints"

            assert wait_for(p(pidRE.search, get_output()))
            pid = int(pidRE.search(get_output()).groups()[0])

            os.kill(pid, signal.SIGTERM)

            time.sleep(3)
            backend.datapoints.clear()

            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "redis_info")), \
                "didn't get datapoints after Python process was killed"

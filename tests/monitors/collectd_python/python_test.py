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
from signalfx.generated_protocol_buffers import signal_fx_protocol_buffers_pb2 as sf_pbuf

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, has_datapoint_with_dim, regex_search_matches_output, tcp_socket_open
from tests.helpers.util import container_ip, run_container, wait_for
from tests.paths import BUNDLE_DIR

pytestmark = [pytest.mark.pyrunner]

PID_RE = re.compile(r"runnerPID=(\d+)")

MONITOR_CONFIG = string.Template(
    """
monitors:
  - type: collectd/python
    moduleName: redis_info
    modulePaths:
     - "${bundle_root}/collectd-python/redis"
    pluginConfig:
      Host: $host
      Port: 6379
      Verbose: true
      Redis_uptime_in_seconds: "gauge"
      Redis_lru_clock: "counter"
"""
)


def test_python_runner_with_redis():
    with run_container("redis:4-alpine") as test_container:
        host = container_ip(test_container)
        config = MONITOR_CONFIG.substitute(host=host, bundle_root=BUNDLE_DIR)
        assert wait_for(p(tcp_socket_open, host, 6379), 60), "redis is not listening on port"

        redis_client = redis.StrictRedis(host=host, port=6379, db=0)
        assert wait_for(redis_client.ping, 60), "service didn't start"

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "redis_info")
            ), "didn't get datapoints"

            assert wait_for(p(regex_search_matches_output, agent.get_output, PID_RE.search))
            pid = int(PID_RE.search(agent.output).groups()[0])

            os.kill(pid, signal.SIGTERM)

            time.sleep(3)
            agent.fake_services.reset_datapoints()

            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "redis_info")
            ), "didn't get datapoints after Python process was killed"

            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="counter.lru_clock",
                    metric_type=sf_pbuf.CUMULATIVE_COUNTER,
                ),
                timeout_seconds=3,
            ), "metric type was wrong"

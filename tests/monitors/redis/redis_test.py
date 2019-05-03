import string
from contextlib import contextmanager
from functools import partial as p
from pathlib import Path
from textwrap import dedent

import pytest
import redis
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.util import container_ip, run_container, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.redis, pytest.mark.monitor_with_endpoints]

MONITOR_CONFIG = string.Template(
    """
monitors:
- type: collectd/redis
  host: $host
  port: 6379
"""
)
SCRIPT_DIR = Path(__file__).parent.resolve()


@contextmanager
def run_redis(image="redis:4-alpine"):
    with run_container(image) as redis_container:
        host = container_ip(redis_container)
        assert wait_for(p(tcp_socket_open, host, 6379), 60), "service not listening on port"

        redis_client = redis.StrictRedis(host=host, port=6379, db=0)
        assert wait_for(redis_client.ping, 60), "service didn't start"

        yield [host, redis_client]


@pytest.mark.parametrize("image", ["redis:3-alpine", "redis:4-alpine"])
def test_redis(image):
    with run_redis(image) as [hostname, _]:
        config = MONITOR_CONFIG.substitute(host=hostname)
        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"plugin": "redis_info"})
            ), "didn't get datapoints"


def test_redis_key_lengths():
    with run_redis() as [hostname, redis_client]:
        redis_client.lpush("queue-1", *["a", "b", "c"])
        redis_client.lpush("queue-2", *["x", "y"])

        config = dedent(
            f"""
          monitors:
           - type: collectd/redis
             host: {hostname}
             port: 6379
             sendListLengths:
              - databaseIndex: 0
                keyPattern: queue-*
        """
        )
        with Agent.run(config) as agent:
            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="gauge.key_llen",
                    dimensions={"key_name": "queue-1"},
                    value=3,
                )
            ), "didn't get datapoints"
            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="gauge.key_llen",
                    dimensions={"key_name": "queue-2"},
                    value=2,
                )
            ), "didn't get datapoints"

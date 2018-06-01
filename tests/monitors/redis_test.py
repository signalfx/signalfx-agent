from functools import partial as p
import pytest
import redis
import string

from tests.helpers.util import wait_for, run_agent, run_container, container_ip
from tests.helpers.assertions import tcp_socket_open, has_datapoint_with_dim

monitor_config = string.Template("""
monitors:
- type: collectd/redis
  host: $host
  port: 6379
  sendListLengths:
  - databaseIndex: 0
    keyPattern: '*'
""")


@pytest.mark.parametrize("image", [
    "redis:3-alpine",
    "redis:4-alpine"
])
def test_redis(image):
    with run_container(image) as test_container:
        host = container_ip(test_container)
        config = monitor_config.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 6379), 60), "service not listening on port"

        redis_client = redis.StrictRedis(host=host, port=6379, db=0)
        assert wait_for(redis_client.ping, 60), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "redis_info")), "didn't get datapoints"

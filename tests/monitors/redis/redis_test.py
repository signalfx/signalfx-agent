import os
import string
from contextlib import contextmanager
from functools import partial as p
from textwrap import dedent

import pytest
import redis

from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test
from tests.helpers.util import (
    container_ip,
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_agent,
    run_container,
    wait_for,
)

pytestmark = [pytest.mark.collectd, pytest.mark.redis, pytest.mark.monitor_with_endpoints]

MONITOR_CONFIG = string.Template(
    """
monitors:
- type: collectd/redis
  host: $host
  port: 6379
"""
)


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
        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint, backend, dimensions={"plugin": "redis_info"})), "didn't get datapoints"


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
        with run_agent(config) as [backend, _, _]:
            assert wait_for(
                p(has_datapoint, backend, metric_name="gauge.key_llen", dimensions={"key_name": "queue-1"}, value=3)
            ), "didn't get datapoints"
            assert wait_for(
                p(has_datapoint, backend, metric_name="gauge.key_llen", dimensions={"key_name": "queue-2"}, value=2)
            ), "didn't get datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_redis_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = os.path.join(os.path.dirname(os.path.realpath(__file__)), "redis-k8s.yaml")
    monitors = [
        {"type": "collectd/redis", "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace)}
    ]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=[yaml],
        observer=k8s_observer,
        expected_metrics=get_monitor_metrics_from_selfdescribe(monitors[0]["type"]),
        expected_dims=get_monitor_dims_from_selfdescribe(monitors[0]["type"]),
        test_timeout=k8s_test_timeout,
    )

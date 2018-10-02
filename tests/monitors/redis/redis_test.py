import os
import string
from functools import partial as p

import pytest
import redis

from helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from helpers.kubernetes.utils import get_discovery_rule, run_k8s_monitors_test
from helpers.util import (
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
  sendListLengths:
  - databaseIndex: 0
    keyPattern: '*'
"""
)


@pytest.mark.parametrize("image", ["redis:3-alpine", "redis:4-alpine"])
def test_redis(image):
    with run_container(image) as test_container:
        host = container_ip(test_container)
        config = MONITOR_CONFIG.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 6379), 60), "service not listening on port"

        redis_client = redis.StrictRedis(host=host, port=6379, db=0)
        assert wait_for(redis_client.ping, 60), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "redis_info")), "didn't get datapoints"


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

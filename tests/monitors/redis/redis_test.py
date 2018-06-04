from functools import partial as p
import os
import pytest
import redis
import string

from tests.helpers.util import wait_for, run_agent, run_container, container_ip
from tests.helpers.assertions import tcp_socket_open, has_datapoint_with_dim
from tests.kubernetes.utils import (
    run_k8s_monitors_test,
    get_metrics_from_doc,
    get_dims_from_doc,
)

pytestmark = [pytest.mark.collectd, pytest.mark.redis, pytest.mark.monitor_with_endpoints]

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


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_redis_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    monitors = [
        {"type": "collectd/redis",
         "discoveryRule": 'container_image =~ "redis" && private_port == 6379 && kubernetes_namespace == "%s"' % k8s_namespace}
    ]
    yamls = [os.path.join(os.path.dirname(os.path.realpath(__file__)), "redis-k8s.yaml")]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        yamls=yamls,
        observer=k8s_observer,
        expected_metrics=get_metrics_from_doc("collectd-redis.md"),
        expected_dims=get_dims_from_doc("collectd-redis.md"),
        test_timeout=k8s_test_timeout)


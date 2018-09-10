from functools import partial as p
import os
import pytest
import string

from tests.helpers.util import (
    wait_for,
    run_agent,
    run_service,
    container_ip,
    get_monitor_metrics_from_selfdescribe,
    get_monitor_dims_from_selfdescribe
)
from tests.kubernetes.utils import (
    tcp_socket_open,
    has_datapoint_with_dim,
    run_k8s_monitors_test,
    get_discovery_rule,
)

pytestmark = [pytest.mark.collectd, pytest.mark.haproxy, pytest.mark.monitor_with_endpoints]


monitor_config = string.Template("""
monitors:
- type: collectd/haproxy
  host: $host
  port: 9000
  enhancedMetrics: true
""")

# @pytest.mark.flaky(reruns=2)
@pytest.mark.parametrize("version", ["latest"])
def test_marathon(version):
    with run_service("haproxy", buildargs={"HAPROXY_VERSION": version}) as service_container:
        host = container_ip(service_container)
        config = monitor_config.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 9000), 120), "haproxy not listening on port"
        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "haproxy")), "didn't get datapoints"

@pytest.mark.k8s
@pytest.mark.kubernetes
def test_haproxy_in_k8s(agent_image, minikube, k8s_observer, k8s_test_timeout, k8s_namespace):
    yaml = os.path.join(os.path.dirname(os.path.realpath(__file__)), "haproxy-k8s.yaml")
    monitors = [
        {"type": "collectd/haproxy",
         "discoveryRule": get_discovery_rule(yaml, k8s_observer, namespace=k8s_namespace),
         "enhancedMetrics": True},
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
        test_timeout=k8s_test_timeout)


"""
Tests for the cadvisor monitor
"""
from functools import partial as p
from textwrap import dedent

import pytest
import semver

from helpers.assertions import any_metric_found, tcp_socket_open
from helpers.kubernetes.utils import run_k8s_monitors_test
from helpers.util import (
    get_monitor_dims_from_selfdescribe,
    get_monitor_metrics_from_selfdescribe,
    run_container,
    container_ip,
    wait_for,
    run_agent,
)

pytestmark = [pytest.mark.cadvisor, pytest.mark.monitor_without_endpoints]


def test_cadvisor():
    cadvisor_opts = dict(
        volumes={
            "/": {"bind": "/rootfs", "mode": "ro"},
            "/var/run": {"bind": "/var/run", "mode": "ro"},
            "/sys": {"bind": "/sys", "mode": "ro"},
            "/var/lib/docker": {"bind": "/var/lib/docker", "mode": "ro"},
            "/dev/disk": {"bind": "/dev/disk", "mode": "ro"},
        }
    )
    with run_container("google/cadvisor:latest", **cadvisor_opts) as cadvisor_container:
        host = container_ip(cadvisor_container)
        config = dedent(
            f"""
            monitors:
              - type: cadvisor
                cadvisorURL: http://{host}:8080
        """
        )
        assert wait_for(p(tcp_socket_open, host, 8080), 60), "service didn't start"
        with run_agent(config) as [backend, _, _]:
            expected_metrics = get_monitor_metrics_from_selfdescribe("cadvisor")
            assert wait_for(p(any_metric_found, backend, expected_metrics)), "Didn't get cadvisor datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_cadvisor_in_k8s(agent_image, minikube, k8s_test_timeout, k8s_namespace):
    if semver.match(minikube.k8s_version.lstrip("v"), ">=1.12.0"):
        pytest.skip("cadvisor web removed from kubelet in v1.12.0")
    monitors = [{"type": "cadvisor", "cadvisorURL": "http://localhost:4194"}]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        expected_metrics=get_monitor_metrics_from_selfdescribe(monitors[0]["type"]),
        expected_dims=get_monitor_dims_from_selfdescribe(monitors[0]["type"]),
        test_timeout=k8s_test_timeout,
    )

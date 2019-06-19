"""
Tests for the cadvisor monitor
"""
from functools import partial as p

import pytest

from tests.helpers.assertions import tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_container, wait_for
from tests.helpers.verify import run_agent_verify

pytestmark = [pytest.mark.cadvisor, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("cadvisor", mon_type="cadvisor")


def run(config, metrics):
    cadvisor_opts = dict(
        volumes={
            "/": {"bind": "/rootfs", "mode": "ro"},
            "/var/run": {"bind": "/var/run", "mode": "ro"},
            "/sys": {"bind": "/sys", "mode": "ro"},
            "/var/lib/docker": {"bind": "/var/lib/docker", "mode": "ro"},
            "/dev/disk": {"bind": "/dev/disk", "mode": "ro"},
        }
    )
    with run_container("google/cadvisor:latest", **cadvisor_opts) as cadvisor_container, run_container(
        # Run container to generate memory limit metric.
        "alpine",
        command=["tail", "-f", "/dev/null"],
        mem_limit="64m",
    ):
        host = container_ip(cadvisor_container)
        assert wait_for(p(tcp_socket_open, host, 8080), 60), "service didn't start"
        run_agent_verify(config.format(host=host), metrics)


def test_cadvisor_default():
    run(
        """
        monitors:
          - type: cadvisor
            cadvisorURL: http://{host}:8080
        """,
        METADATA.default_metrics,
    )


def test_cadvisor_all():
    run(
        """
        monitors:
          - type: cadvisor
            cadvisorURL: http://{host}:8080
            extraMetrics: ["*"]
        """,
        METADATA.all_metrics,
    )

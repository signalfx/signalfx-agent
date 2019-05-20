import time
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, has_datapoint_with_metric_name
from tests.helpers.metadata import Metadata
from tests.helpers.util import ensure_always, run_service, wait_for
from tests.helpers.verify import run_agent_verify_included_metrics, verify_expected_is_subset

pytestmark = [pytest.mark.docker_container_stats, pytest.mark.monitor_without_endpoints]


def test_docker_container_stats():
    with run_service("nginx") as nginx_container:
        with Agent.run(
            """
    monitors:
      - type: docker-container-stats
        enableExtraCPUMetrics: true
        enableExtraMemoryMetrics: true
    """
        ) as agent:
            assert wait_for(
                p(has_datapoint_with_metric_name, agent.fake_services, "cpu.percent")
            ), "Didn't get docker cpu datapoints"
            assert wait_for(
                p(has_datapoint_with_metric_name, agent.fake_services, "memory.percent")
            ), "Didn't get docker memory datapoints"
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "container_id", nginx_container.id)
            ), "Didn't get nginx datapoints"


def test_docker_image_filtering():
    with run_service("nginx") as nginx_container:
        with Agent.run(
            """
    monitors:
      - type: docker-container-stats
        excludedImages:
         - "%s"

    """
            % nginx_container.attrs["Image"]
        ) as agent:
            assert ensure_always(
                lambda: not has_datapoint_with_dim(agent.fake_services, "container_id", nginx_container.id)
            )


def test_docker_label_dimensions():
    with run_service("nginx", labels={"app": "myserver"}):
        with Agent.run(
            """
    monitors:
      - type: docker-container-stats
        labelsToDimensions:
          app: service

    """
        ) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "service", "myserver")
            ), "Didn't get datapoint with service dim"


def test_docker_envvar_dimensions():
    with run_service("nginx", environment={"APP": "myserver"}):
        with Agent.run(
            """
    monitors:
      - type: docker-container-stats
        envToDimensions:
          APP: app

    """
        ) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "app", "myserver")
            ), "Didn't get datapoint with service app"


def test_docker_detects_new_containers():
    with Agent.run(
        """
    monitors:
      - type: docker-container-stats

    """
    ) as agent:
        time.sleep(5)
        with run_service("nginx") as nginx_container:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "container_id", nginx_container.id)
            ), "Didn't get nginx datapoints"


def test_docker_stops_watching_paused_containers():
    with run_service("nginx") as nginx_container:
        with Agent.run(
            """
        monitors:
          - type: docker-container-stats

        """
        ) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "container_id", nginx_container.id)
            ), "Didn't get nginx datapoints"
            nginx_container.pause()
            time.sleep(5)
            agent.fake_services.reset_datapoints()
            assert ensure_always(
                lambda: not has_datapoint_with_dim(agent.fake_services, "container_id", nginx_container.id)
            )


def test_docker_stops_watching_stopped_containers():
    with run_service("nginx") as nginx_container:
        with Agent.run(
            """
        monitors:
          - type: docker-container-stats

        """
        ) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "container_id", nginx_container.id)
            ), "Didn't get nginx datapoints"
            nginx_container.stop(timeout=10)
            time.sleep(5)
            agent.fake_services.reset_datapoints()
            assert ensure_always(
                lambda: not has_datapoint_with_dim(agent.fake_services, "container_id", nginx_container.id)
            )


def test_docker_stops_watching_destroyed_containers():
    with run_service("nginx") as nginx_container:
        with Agent.run(
            """
        monitors:
          - type: docker-container-stats

        """
        ) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "container_id", nginx_container.id)
            ), "Didn't get nginx datapoints"
            nginx_container.remove(force=True)
            time.sleep(5)
            agent.fake_services.reset_datapoints()
            assert ensure_always(
                lambda: not has_datapoint_with_dim(agent.fake_services, "container_id", nginx_container.id)
            )


METADATA = Metadata.from_package("docker")


def test_docker_included():
    with run_service(
        "elasticsearch/6.6.1"
    ):  # just get a container that does some block io running so we have some stats
        run_agent_verify_included_metrics(
            f"""
            monitors:
            - type: docker-container-stats
            """,
            METADATA,
        )


ENHANCED_METRICS = METADATA.all_metrics - {
    "memory.stats.swap",
    # The following metrics are inconsistent across docker versions
    "blkio.io_time_recursive.async",
    "blkio.io_time_recursive.sync",
    "blkio.io_time_recursive.write",
    "blkio.io_time_recursive.total",
    "blkio.io_time_recursive.read",
    "blkio.io_queue_recursive.total",
    "blkio.io_service_time_recursive.read",
    "blkio.io_wait_time_recursive.sync",
    "blkio.io_merged_recursive.sync",
    "blkio.io_wait_time_recursive.total",
    "blkio.io_queue_recursive.read",
    "blkio.io_service_time_recursive.write",
    "blkio.io_service_time_recursive.total",
    "blkio.io_merged_recursive.write",
    "blkio.io_wait_time_recursive.async",
    "blkio.io_queue_recursive.async",
    "blkio.io_wait_time_recursive.read",
    "blkio.io_service_time_recursive.sync",
    "blkio.io_merged_recursive.total",
    "blkio.io_service_time_recursive.async",
    "blkio.io_queue_recursive.sync",
    "blkio.io_merged_recursive.async",
    "blkio.io_merged_recursive.read",
    "blkio.io_wait_time_recursive.write",
    "blkio.io_queue_recursive.write",
}


def test_docker_enhanced():
    with run_service(
        "elasticsearch/6.6.1"
    ):  # just get a container that does some block io running so we have some stats
        with Agent.run(
            f"""
            monitors:
            - type: docker-container-stats
              enableExtraBlockIOMetrics: true
              enableExtraCPUMetrics: true
              enableExtraMemoryMetrics: true
              enableExtraNetworkMetrics: true
            """
        ) as agent:
            verify_expected_is_subset(agent, ENHANCED_METRICS)

from functools import partial as p
import os
import pytest
import string
import time

from tests.helpers.util import ensure_always, wait_for, run_agent, run_service
from tests.helpers.assertions import *

from tests.kubernetes.utils import (
    run_k8s_monitors_test,
    get_metrics_from_doc,
    get_dims_from_doc,
)

pytestmark = [pytest.mark.docker_container_stats, pytest.mark.monitor_without_endpoints]


def test_docker_container_stats():
    with run_service("nginx") as nginx_container:
        with run_agent("""
    monitors:
      - type: docker-container-stats

    """) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_metric_name, backend, "cpu.percent")), "Didn't get docker datapoints"
            assert wait_for(p(has_datapoint_with_dim, backend, "container_id", nginx_container.id)), "Didn't get nginx datapoints"


def test_docker_image_filtering():
    with run_service("nginx") as nginx_container:
        with run_agent("""
    monitors:
      - type: docker-container-stats
        excludedImages:
         - "%s"

    """ % nginx_container.attrs["Image"]) as [backend, _, _]:
            assert ensure_always(lambda: not has_datapoint_with_dim(backend, "container_id", nginx_container.id))


def test_docker_label_dimensions():
    with run_service("nginx", labels={"app": "myserver"}) as nginx_container:
        with run_agent("""
    monitors:
      - type: docker-container-stats
        labelsToDimensions:
          app: service

    """) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "service", "myserver")), "Didn't get datapoint with service dim"

def test_docker_detects_new_containers():
    with run_agent("""
    monitors:
      - type: docker-container-stats

    """) as [backend, _, _]:
        time.sleep(5)
        with run_service("nginx") as nginx_container:
            assert wait_for(p(has_datapoint_with_dim, backend, "container_id", nginx_container.id)), "Didn't get nginx datapoints"

def test_docker_stops_watching_old_containers():
    with run_service("nginx") as nginx_container:
        with run_agent("""
        monitors:
          - type: docker-container-stats

        """) as [backend, get_output, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "container_id", nginx_container.id)), "Didn't get nginx datapoints"
            nginx_container.stop(timeout=10)
            time.sleep(3)
            backend.datapoints.clear()
            assert ensure_always(lambda: not has_datapoint_with_dim(backend, "container_id", nginx_container.id))


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_docker_container_stats_in_k8s(agent_image, minikube, k8s_test_timeout, k8s_namespace):
    monitors = [
        {"type": "docker-container-stats"}
    ]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        expected_metrics=get_metrics_from_doc("docker-container-stats.md"),
        expected_dims=get_dims_from_doc("docker-container-stats.md"),
        test_timeout=k8s_test_timeout)


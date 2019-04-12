"""
Integration tests for the ecs metadata monitor
"""
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, has_datapoint_with_metric_name
from tests.helpers.util import container_ip, ensure_always, run_container, run_service, wait_for

pytestmark = [pytest.mark.docker_container_stats, pytest.mark.monitor_without_endpoints]


def test_ecs_container_stats():
    with run_service("ecsmeta") as ecsmeta, run_container("redis:4-alpine") as redis:
        ecsmeta_ip = container_ip(ecsmeta)
        redis_ip = container_ip(redis)
        with Agent.run(
            """
    monitors:
      - type: ecs-metadata
        enableExtraCPUMetrics: true
        enableExtraMemoryMetrics: true
        metadataEndpoint: http://%s/metadata_single?redis_ip=%s
        statsEndpoint: http://%s/stats

    """
            % (ecsmeta_ip, redis_ip, ecsmeta_ip)
        ) as agent:
            assert wait_for(
                p(has_datapoint_with_metric_name, agent.fake_services, "cpu.percent")
            ), "Didn't get docker cpu datapoints"
            assert wait_for(
                p(has_datapoint_with_metric_name, agent.fake_services, "memory.percent")
            ), "Didn't get docker memory datapoints"
            assert wait_for(
                # container_id is included in stats.json file in ecsmeta app
                # because stats data don't come directly from the docker container but from ecs metadata api
                p(
                    has_datapoint_with_dim,
                    agent.fake_services,
                    "container_id",
                    "c42fa5a73634bcb6e301dfb7b13ac7ead2af473210be6a15da75a290c283b66c",
                )
            ), "Didn't get redis datapoints"


def test_ecs_container_image_filtering():
    with run_service("ecsmeta") as ecsmeta, run_container("redis:4-alpine") as redis:
        ecsmeta_ip = container_ip(ecsmeta)
        redis_ip = container_ip(redis)
        with Agent.run(
            """
    monitors:
      - type: ecs-metadata
        metadataEndpoint: http://%s/metadata_single?redis_ip=%s
        statsEndpoint: http://%s/stats
        excludedImages:
          - redis:latest

    """
            % (ecsmeta_ip, redis_ip, ecsmeta_ip)
        ) as agent:
            assert ensure_always(
                lambda: not has_datapoint_with_dim(
                    agent.fake_services,
                    "container_id",
                    "c42fa5a73634bcb6e301dfb7b13ac7ead2af473210be6a15da75a290c283b66c",
                )
            )


def test_ecs_container_label_dimension():
    with run_service("ecsmeta") as ecsmeta, run_container("redis:4-alpine") as redis:
        ecsmeta_ip = container_ip(ecsmeta)
        redis_ip = container_ip(redis)
        with Agent.run(
            """
    monitors:
      - type: ecs-metadata
        metadataEndpoint: http://%s/metadata_single?redis_ip=%s
        statsEndpoint: http://%s/stats
        labelsToDimensions:
          container_name: container_title

    """
            % (ecsmeta_ip, redis_ip, ecsmeta_ip)
        ) as agent:
            assert ensure_always(
                lambda: not has_datapoint_with_dim(
                    agent.fake_services, "container_title", "ecs-seon-fargate-test-3-redis-baf2cfda88f8d8ee4900"
                )
            )


def test_ecs_container_stats_without_container_metadata():
    with run_service("ecsmeta") as ecsmeta, run_container("redis:4-alpine") as redis:
        ecsmeta_ip = container_ip(ecsmeta)
        redis_ip = container_ip(redis)
        with Agent.run(
            """
    monitors:
      - type: ecs-metadata
        metadataEndpoint: http://%s/metadata_single?redis_ip=%s&mask_redis=true
        statsEndpoint: http://%s/stats

    """
            % (ecsmeta_ip, redis_ip, ecsmeta_ip)
        ) as agent:
            assert wait_for(
                # container_id is included in stats.json file in ecsmeta app
                # because stats data don't come directly from the docker container but from ecs metadata api
                p(
                    has_datapoint_with_dim,
                    agent.fake_services,
                    "container_id",
                    "c42fa5a73634bcb6e301dfb7b13ac7ead2af473210be6a15da75a290c283b66c",
                )
            ), "Didn't get redis datapoints"

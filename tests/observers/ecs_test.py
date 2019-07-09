"""
Integration tests for the ecs observer
"""
import string
import time
from functools import partial as p

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.util import container_ip, ensure_always, run_container, run_service, wait_for

CONFIG = string.Template(
    """
observers:
  - type: ecs
    metadataEndpoint: http://$host/$case?redis_ip=$redis_ip&mongo_ip=$mongo_ip
    labelsToDimensions:
      com.amazonaws.ecs.container-name: container_spec_name

monitors:
  - type: collectd/redis
    discoveryRule: container_image =~ "redis" && port == 6379
  - type: collectd/mongodb
    discoveryRule: container_image =~ "mongo" && port == 27017
"""
)


def test_ecs_observer_single():
    with run_service("ecsmeta") as ecsmeta:
        with run_container("redis:4-alpine") as redis:
            with Agent.run(
                CONFIG.substitute(
                    host=container_ip(ecsmeta),
                    redis_ip=container_ip(redis),
                    mongo_ip="not_used",
                    case="metadata_single",
                )
            ) as agent:
                assert wait_for(
                    p(has_datapoint_with_dim, agent.fake_services, "container_image", "redis:latest")
                ), "Didn't get redis datapoints"

            # Let redis be removed by docker observer and collectd restart
            time.sleep(5)
            agent.fake_services.datapoints.clear()
            assert ensure_always(
                lambda: not has_datapoint_with_dim(agent.fake_services, "ClusterName", "seon-fargate-test"), 10
            )


def test_ecs_observer_multi_containers():
    with run_service("ecsmeta") as ecsmeta:
        with run_container("redis:4-alpine") as redis, run_container("mongo:4") as mongo:
            with Agent.run(
                CONFIG.substitute(
                    host=container_ip(ecsmeta),
                    redis_ip=container_ip(redis),
                    mongo_ip=container_ip(mongo),
                    case="metadata_multi_containers",
                )
            ) as agent:
                assert wait_for(
                    p(has_datapoint_with_dim, agent.fake_services, "container_image", "redis:latest")
                ), "Didn't get redis datapoints"
                assert wait_for(
                    p(has_datapoint_with_dim, agent.fake_services, "container_image", "mongo:latest")
                ), "Didn't get mongo datapoints"

            # Let redis be removed by docker observer and collectd restart
            time.sleep(5)
            agent.fake_services.datapoints.clear()
            assert ensure_always(
                lambda: not has_datapoint_with_dim(agent.fake_services, "ClusterName", "seon-fargate-test"), 10
            )


def test_ecs_observer_dim_label():
    with run_service("ecsmeta") as ecsmeta:
        with run_container("redis:4-alpine") as redis:
            with Agent.run(
                CONFIG.substitute(
                    host=container_ip(ecsmeta),
                    redis_ip=container_ip(redis),
                    mongo_ip="not_used",
                    case="metadata_single",
                )
            ) as agent:
                assert wait_for(
                    p(has_datapoint_with_dim, agent.fake_services, "container_spec_name", "redis")
                ), "Didn't get redis datapoints"

            # Let redis be removed by docker observer and collectd restart
            time.sleep(5)
            agent.fake_services.datapoints.clear()
            assert ensure_always(
                lambda: not has_datapoint_with_dim(agent.fake_services, "ClusterName", "seon-fargate-test"), 10
            )

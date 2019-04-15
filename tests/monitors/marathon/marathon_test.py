from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, http_status, tcp_socket_open
from tests.helpers.util import container_ip, run_container, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.marathon, pytest.mark.monitor_with_endpoints]


@pytest.mark.flaky(reruns=2)
@pytest.mark.parametrize("marathon_image", ["mesosphere/marathon:v1.1.1", "mesosphere/marathon:v1.6.352"])
def test_marathon(marathon_image):
    with run_container("zookeeper:3.5") as zookeeper:
        zkhost = container_ip(zookeeper)
        assert wait_for(p(tcp_socket_open, zkhost, 2181), 60), "zookeeper didn't start"
        with run_container(
            marathon_image, command=["--master", "localhost:5050", "--zk", "zk://{0}:2181/marathon".format(zkhost)]
        ) as service_container:
            host = container_ip(service_container)
            config = dedent(
                f"""
                monitors:
                - type: collectd/marathon
                  host: {host}
                  port: 8080
                """
            )

            assert wait_for(p(tcp_socket_open, host, 8080), 120), "marathon not listening on port"
            assert wait_for(
                p(http_status, url="http://{0}:8080/v2/info".format(host), status=[200]), 120
            ), "service didn't start"

            with Agent.run(config) as agent:
                assert wait_for(
                    p(has_datapoint_with_dim, agent.fake_services, "plugin", "marathon")
                ), "didn't get datapoints"

from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, http_status, tcp_socket_open
from tests.helpers.util import container_ip, run_service, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.jenkins, pytest.mark.monitor_with_endpoints]

METRICS_KEY = "33DD8B2F1FD645B814993275703F_EE1FD4D4E204446D5F3200E0F6-C55AC14E"


@pytest.mark.flaky(reruns=2)
@pytest.mark.parametrize(
    "version",
    [
        # technically we support 1.580.3, but the scripts needed to programmatically
        # setup jenkins do not work prior to 1.651.3
        "1.651.3-alpine",
        # TODO: jenkins doesn't have a latest tag so we'll need to update this
        # periodically
        "2.60.3-alpine",
    ],
)
def test_jenkins(version):
    with run_service("jenkins", buildargs={"JENKINS_VERSION": version, "JENKINS_PORT": "8080"}) as jenkins_container:
        host = container_ip(jenkins_container)
        config = dedent(
            f"""
            monitors:
              - type: collectd/jenkins
                host: {host}
                port: 8080
                metricsKey: {METRICS_KEY}
            """
        )
        assert wait_for(p(tcp_socket_open, host, 8080), 60), "service not listening on port"
        assert wait_for(
            p(http_status, url=f"http://{host}:8080/metrics/{METRICS_KEY}/ping/", status=[200]), 120
        ), "service didn't start"

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "jenkins")
            ), "Didn't get jenkins datapoints"

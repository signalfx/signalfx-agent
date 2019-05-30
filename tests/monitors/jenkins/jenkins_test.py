from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, http_status, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for
from tests.helpers.verify import run_agent_verify, verify_expected_is_subset

pytestmark = [pytest.mark.collectd, pytest.mark.jenkins, pytest.mark.monitor_with_endpoints]

METRICS_KEY = "33DD8B2F1FD645B814993275703F_EE1FD4D4E204446D5F3200E0F6-C55AC14E"

JENKINS_VERSIONS = [
    # technically we support 1.580.3, but the scripts needed to programmatically
    # setup jenkins do not work prior to 1.651.3
    "1.651.3-alpine",
    # TODO: jenkins doesn't have a latest tag so we'll need to update this
    # periodically
    "2.60.3-alpine",
]

METADATA = Metadata.from_package("collectd/jenkins")
ENHANCED_METRICS = {
    "1.651.3-alpine": METADATA.all_metrics
    - {
        "gauge.jenkins.job.duration",
        "gauge.jenkins.node.executor.count.value",
        "gauge.jenkins.node.executor.in-use.value",
        "gauge.jenkins.node.health-check.score",
        "gauge.jenkins.node.queue.size.value",
        "gauge.jenkins.node.slave.online.status",
        "gauge.jenkins.node.vm.memory.heap.usage",
        "gauge.jenkins.node.vm.memory.non-heap.used",
        "gauge.jenkins.node.vm.memory.total.used",
    },
    "2.60.3-alpine": METADATA.all_metrics - {"gauge.jenkins.job.duration", "gauge.jenkins.node.slave.online.status"},
}


@pytest.mark.flaky(reruns=2)
@pytest.mark.parametrize("version", JENKINS_VERSIONS)
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


@pytest.mark.flaky(reruns=2)
@pytest.mark.parametrize("version", JENKINS_VERSIONS)
def test_jenkins_default(version):
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

        run_agent_verify(config, ENHANCED_METRICS[version])


@pytest.mark.flaky(reruns=2)
@pytest.mark.parametrize("version", JENKINS_VERSIONS)
def test_jenkins_enhanced(version):
    with run_service("jenkins", buildargs={"JENKINS_VERSION": version, "JENKINS_PORT": "8080"}) as jenkins_container:
        host = container_ip(jenkins_container)
        config = dedent(
            f"""
            monitors:
              - type: collectd/jenkins
                host: {host}
                port: 8080
                metricsKey: {METRICS_KEY}
                enhancedMetrics: true
            """
        )
        assert wait_for(p(tcp_socket_open, host, 8080), 60), "service not listening on port"
        assert wait_for(
            p(http_status, url=f"http://{host}:8080/metrics/{METRICS_KEY}/ping/", status=[200]), 120
        ), "service didn't start"

        with Agent.run(config) as agent:
            verify_expected_is_subset(agent, ENHANCED_METRICS[version])

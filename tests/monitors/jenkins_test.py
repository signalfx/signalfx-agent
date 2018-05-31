from functools import partial as p
import string
import pytest

from tests.helpers.util import wait_for, run_agent, run_service, container_ip
from tests.helpers.assertions import tcp_socket_open, has_datapoint_with_dim, http_status

jenkins_config = string.Template("""
monitors:
  - type: collectd/jenkins
    host: $host
    port: $port
    metricsKey: 33DD8B2F1FD645B814993275703F_EE1FD4D4E204446D5F3200E0F6-C55AC14E
""")


@pytest.mark.parametrize("version", [
    # technically we support 1.580.3, but the scripts needed to programmatically
    # setup jenkins do not work prior to 1.651.3
    "1.651.3-alpine",
    # TODO: jenkins doesn't have a latest tag so we'll need to update this
    # periodically
    "2.60.3-alpine"
])
def test_jenkins(version):
    with run_service("jenkins", buildargs={"JENKINS_VERSION": version, "JENKINS_PORT": "8080"}) as jenkins_container:
        host = container_ip(jenkins_container)
        config = jenkins_config.substitute(host=host, port=8080)
        assert wait_for(p(tcp_socket_open, host, 8080), 60), "service not listening on port"
        assert wait_for(p(http_status, 200, "http://{0}:8080/metrics/33DD8B2F1FD645B814993275703F_EE1FD4D4E204446D5F3200E0F6-C55AC14E/ping/".format(host)), 120), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "jenkins")), "Didn't get jenkins datapoints"

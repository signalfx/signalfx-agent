from functools import partial as p
import pytest
import string

from tests.helpers.util import wait_for, run_agent, run_container, container_ip
from tests.helpers.assertions import tcp_socket_open, has_datapoint_with_dim, http_status

monitor_config = string.Template("""
monitors:
- type: collectd/marathon
  host: $host
  port: 8080
""")


@pytest.mark.parametrize("image", [
    "mesosphere/marathon:v1.1.1",
    "mesosphere/marathon:v1.6.352"
])
def test_marathon(image):
    with run_container("zookeeper:3.5") as zookeeper:
        zkhost = container_ip(zookeeper)
        assert wait_for(p(tcp_socket_open, zkhost, 2181), 60), "zookeeper didn't start"
        with run_container(image,
                           command=["--master", "localhost:5050", "--zk", "zk://{0}:2181/marathon".format(zkhost)]
                           ) as service_container:
            host = container_ip(service_container)
            config = monitor_config.substitute(host=host)
            assert wait_for(p(tcp_socket_open, host, 8080), 60), "service not listening on port"
            assert wait_for(p(http_status, 200, "http://{0}:8080/v2/info".format(host)), 120), "service didn't start"

            with run_agent(config) as [backend, _, _]:
                assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "marathon")), "didn't get datapoints"

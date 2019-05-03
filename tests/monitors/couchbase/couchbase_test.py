import string
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, http_status, tcp_socket_open
from tests.helpers.util import container_ip, run_service, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.couchbase, pytest.mark.monitor_with_endpoints]

COUCHBASE_CONFIG = string.Template(
    """
monitors:
  - type: collectd/couchbase
    host: $host
    port: 8091
    collectTarget: NODE
    username: administrator
    password: password
"""
)


@pytest.mark.flaky(reruns=2)
@pytest.mark.parametrize("tag", ["enterprise-4.0.0", "enterprise-5.1.0"])
def test_couchbase(tag):
    with run_service(
        "couchbase", buildargs={"COUCHBASE_VERSION": tag}, hostname="node1.cluster"
    ) as couchbase_container:
        host = container_ip(couchbase_container)
        config = COUCHBASE_CONFIG.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 8091), 60), "service not listening on port"
        assert wait_for(
            p(
                http_status,
                url=f"http://{host}:8091/pools/default",
                status=[200],
                username="administrator",
                password="password",
            ),
            120,
        ), "service didn't start"

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "couchbase")
            ), "Didn't get couchbase datapoints"

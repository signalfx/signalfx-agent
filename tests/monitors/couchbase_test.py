from functools import partial as p
import pytest
import string

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, run_agent, run_service, container_ip
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open, http_status

couchbase_config = string.Template("""
monitors:
  - type: collectd/couchbase
    host: $host
    port: 8091
    collectTarget: NODE
    username: administrator
    password: password
""")


@pytest.mark.parametrize("tag", [
    "enterprise-4.0.0",
    "enterprise-5.1.0"
])
def test_couchbase(tag):
    with run_service("couchbase",
                     buildargs={"COUCHBASE_VERSION": tag},
                     hostname="node1.cluster") as couchbase_container:
        host = container_ip(couchbase_container)
        config = couchbase_config.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 8091), 60), "service not listening on port"
        assert wait_for(p(http_status, status=[401],
                          url="http://{0}:8091/pools".format(host)),
                        120), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "couchbase")), \
                   "Didn't get couchbase datapoints"

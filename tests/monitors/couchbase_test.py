from functools import partial as p
import string

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, run_agent, run_service
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open

couchbase_config = string.Template("""
monitors:
  - type: collectd/couchbase
    host: $host
    port: 8091
    collectTarget: NODE
    username: administrator
    password: password
""")


def test_couchbase():
    with run_service("couchbase", hostname="node1.cluster") as couchbase_container:
        host_addr = couchbase_container.attrs["NetworkSettings"]["IPAddress"]
        config = couchbase_config.substitute(host=host_addr)
        assert wait_for(p(tcp_socket_open, host_addr, 8091), 60), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "couchbase")), \
                   "Didn't get couchbase datapoints"

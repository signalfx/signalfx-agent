from functools import partial as p
import os
import string

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, run_agent, run_container
from tests.helpers.assertions import *


ETCD_COMMAND="-listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 -advertise-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001"

etcd_config = string.Template("""
monitors:
  - type: collectd/etcd
    host: $host
    port: 2379
    clusterName: test-cluster
""")

def test_etcd_monitor():
    with run_container("quay.io/coreos/etcd:v2.3.8", command=ETCD_COMMAND) as etcd_cont:
        host_addr = etcd_cont.attrs["NetworkSettings"]["IPAddress"]
        config = etcd_config.substitute(host=host_addr)
        assert wait_for(p(tcp_socket_open, host_addr, 2379), 60), "service didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "etcd")), "Didn't get etcd datapoints"


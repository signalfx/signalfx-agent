import string
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.util import container_ip, run_container, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.etcd, pytest.mark.monitor_with_endpoints]

ETCD_COMMAND = (
    "-listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 "
    "-advertise-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001"
)

ETCD_CONFIG = string.Template(
    """
monitors:
  - type: collectd/etcd
    host: $host
    port: 2379
    clusterName: test-cluster
"""
)


def test_etcd_monitor():
    with run_container("quay.io/coreos/etcd:v2.3.8", command=ETCD_COMMAND) as etcd_cont:
        host = container_ip(etcd_cont)
        config = ETCD_CONFIG.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 2379), 60), "service didn't start"

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "etcd")
            ), "Didn't get etcd datapoints"

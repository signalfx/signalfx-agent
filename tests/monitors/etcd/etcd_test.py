import string
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_container, wait_for
from tests.helpers.verify import verify_custom, verify_expected_is_subset

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


METADATA = Metadata.from_package("collectd/etcd")
INCLUDED_METRICS = METADATA.included_metrics - {
    # leader metrics don't occur on this test environment
    "counter.etcd.leader.counts.success",
    "gauge.etcd.leader.latency.current",
    "counter.etcd.leader.counts.fail",
    "gauge.etcd.leader.latency.max",
    "gauge.etcd.leader.latency.average",
    "gauge.etcd.leader.latency.stddev",
    "gauge.etcd.leader.latency.min",
}


def test_etcd_monitor_included():
    with run_container("quay.io/coreos/etcd:v2.3.8", command=ETCD_COMMAND) as etcd_cont:
        host = container_ip(etcd_cont)
        config = ETCD_CONFIG.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 2379), 60), "service didn't start"

        verify_custom(config, INCLUDED_METRICS)


ENHANCED_METRICS = METADATA.all_metrics - {
    # leader metrics don't occur on this test environment
    "counter.etcd.leader.counts.success",
    "gauge.etcd.leader.latency.current",
    "counter.etcd.leader.counts.fail",
    "gauge.etcd.leader.latency.max",
    "gauge.etcd.leader.latency.average",
    "gauge.etcd.leader.latency.stddev",
    "gauge.etcd.leader.latency.min",
}


def test_etcd_monitor_enhanced():
    with run_container("quay.io/coreos/etcd:v2.3.8", command=ETCD_COMMAND) as etcd_cont:
        host = container_ip(etcd_cont)
        config = string.Template(
            """
        monitors:
        - type: collectd/etcd
          host: $host
          port: 2379
          clusterName: test-cluster
          enhancedMetrics: true
        """
        ).substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 2379), 60), "service didn't start"

        with Agent.run(config) as agent:
            verify_expected_is_subset(agent, ENHANCED_METRICS)

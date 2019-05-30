from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_container, wait_for, wait_for_assertion
from tests.helpers.verify import verify

pytestmark = [pytest.mark.collectd, pytest.mark.consul, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("collectd/consul")

EXPECTED_DEFAULTS = METADATA.default_metrics - {
    # We don't get these with the test service of consul for
    # some reason, maybe investigate why.
    "gauge.consul.network.node.latency.avg",
    "gauge.consul.serf.queue.Event.avg",
    "gauge.consul.serf.member.left",
    "gauge.consul.consul.leader.reconcile.avg",
    "gauge.consul.network.node.latency.max",
    "gauge.consul.network.node.latency.min",
    "gauge.consul.raft.leader.lastContact.avg",
    "gauge.consul.raft.leader.lastContact.max",
    "gauge.consul.serf.queue.Event.max",
    "gauge.consul.raft.leader.lastContact.min",
    "gauge.consul.network.dc.latency.avg",
}


def test_consul_defaults():
    with run_container("consul:1.4.4") as consul_cont:
        host = container_ip(consul_cont)
        assert wait_for(p(tcp_socket_open, host, 8500), 60), "consul service didn't start"

        with Agent.run(
            f"""
         monitors:
           - type: collectd/consul
             host: {host}
             port: 8500
             enhancedMetrics: false
         """
        ) as agent:
            verify(agent, EXPECTED_DEFAULTS)


def test_consul_enhanced():
    with run_container("consul:1.4.4") as consul_cont:
        host = container_ip(consul_cont)
        assert wait_for(p(tcp_socket_open, host, 8500), 60), "consul service didn't start"

        with Agent.run(
            f"""
         monitors:
           - type: collectd/consul
             host: {host}
             port: 8500
             enhancedMetrics: true
         """
        ) as agent:
            target_metric = "gauge.consul.serf.events.consul:new-leader"
            assert target_metric in METADATA.nondefault_metrics

            def test():
                assert has_datapoint(agent.fake_services, metric_name=target_metric)

            wait_for_assertion(test)

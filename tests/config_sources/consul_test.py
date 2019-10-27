from contextlib import contextmanager
from functools import partial as p

import consul
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, has_no_datapoint, tcp_socket_open
from tests.helpers.util import container_ip, ensure_always, run_container, wait_for


@contextmanager
def run_consul():
    with run_container("consul:1.6.0") as cont:
        consul_ip = container_ip(cont)
        assert wait_for(p(tcp_socket_open, consul_ip, 8500), 30)
        yield [consul.Consul(host=consul_ip), cont]


def test_basic_consul():
    with run_consul() as [client, cont]:
        client.kv.put("signalfx/env", "test")
        with Agent.run(
            f"""
            intervalSeconds: 2
            globalDimensions:
              env: {{"#from": "consul:signalfx/env"}}
            configSources:
              consul:
                endpoint: {container_ip(cont)}:8500
            monitors:
             - type: collectd/uptime
        """
        ) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"env": "test"}))


def test_optional_with_consul_outage():
    with run_consul() as [client, cont]:
        client.kv.put("signalfx/env", "test")
        cont.pause()

        with Agent.run(
            f"""
            intervalSeconds: 2
            globalDimensions:
              env: {{"#from": "consul:signalfx/env", optional: true}}
            configSources:
              consul:
                endpoint: {container_ip(cont)}:8500
            monitors:
             - type: collectd/uptime
        """
        ) as agent:
            assert wait_for(lambda: agent.fake_services.datapoints)
            assert has_no_datapoint(agent.fake_services, dimensions={"env": "test"})

            cont.unpause()

            assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"env": "test"}))


def test_non_optional_with_consul_outage():
    with run_consul() as [client, cont]:
        client.kv.put("signalfx/env", "test")
        cont.pause()

        with Agent.run(
            f"""
            intervalSeconds: 2
            globalDimensions:
              env: {{"#from": "consul:signalfx/env"}}
            configSources:
              consul:
                endpoint: {container_ip(cont)}:8500
            monitors:
             - type: collectd/uptime
        """
        ) as agent:
            assert ensure_always(lambda: not agent.fake_services.datapoints)

            assert not agent.is_running

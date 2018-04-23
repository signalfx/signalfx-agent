from functools import partial as p
import os
import string

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, run_agent, run_container
from tests.helpers.assertions import *

def wait_for_rabbit_to_start(cont):
    # 3D38 is 15672 in hex, the port we need to be listening
    assert wait_for(p(container_cmd_exit_0, cont, "sh -c 'cat /proc/net/tcp | grep 3D38'")), "rabbitmq didn't start"


rabbitmq_config = string.Template("""
monitors:
  - type: collectd/rabbitmq
    host: $host
    port: 15672
    username: guest
    password: guest
    collectNodes: true
    collectChannels: true
""")

def test_rabbitmq():
    with run_container("rabbitmq:3.6-management") as rabbitmq_cont:
        host = rabbitmq_cont.attrs["NetworkSettings"]["IPAddress"]
        config = rabbitmq_config.substitute(host=host)
        wait_for_rabbit_to_start(rabbitmq_cont)

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "rabbitmq")), "Didn't get rabbitmq datapoints"
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin_instance", "%s-15672" % host)), \
                "Didn't get expected plugin_instance dimension"

def test_rabbitmq_broker_name():
    with run_container("rabbitmq:3.6-management") as rabbitmq_cont:
        host = rabbitmq_cont.attrs["NetworkSettings"]["IPAddress"]
        config = rabbitmq_config.substitute(host=host)
        wait_for_rabbit_to_start(rabbitmq_cont)

        with run_agent("""
monitors:
  - type: collectd/rabbitmq
    host: %s
    brokerName: '{{.host}}-{{.username}}'
    port: 15672
    username: guest
    password: guest
    collectNodes: true
    collectChannels: true
        """ % (host,)) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin_instance", "%s-guest" % host)), \
                "Didn't get expected plugin_instance dimension"

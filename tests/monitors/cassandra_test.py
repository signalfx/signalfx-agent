from functools import partial as p
import os
import string

from tests.helpers import fake_backend
from tests.helpers.util import wait_for, run_agent, run_service
from tests.helpers.assertions import *

cassandra_config = string.Template("""
monitors:
  - type: collectd/cassandra
    host: $host
    port: 7199
    username: cassandra
    password: cassandra
""")

def test_cassandra():
    with run_service("cassandra") as cassandra_cont:
        config = cassandra_config.substitute(host=cassandra_cont.attrs["NetworkSettings"]["IPAddress"])

        # Wait for the JMX port to be open in the container
        assert wait_for(p(container_cmd_exit_0, cassandra_cont,
                        "sh -c 'cat /proc/net/tcp | grep 1C1F'")), "Cassandra JMX didn't start"

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_metric_name, backend, "counter.cassandra.ClientRequest.Read.Latency.Count"), 30), "Didn't get Cassandra datapoints"


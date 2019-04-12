import string
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.util import container_ip, run_container, wait_for
from tests.monitors.hadoop.hadoop_test import start_hadoop

pytestmark = [pytest.mark.collectd, pytest.mark.hadoopjmx, pytest.mark.monitor_with_endpoints]

NODETYPE_PORT = {"nameNode": 5677, "dataNode": 5677, "resourceManager": 5680, "nodeManager": 8002}
YARN_VAR = {"resourceManager": "YARN_RESOURCEMANAGER_OPTS", "nodeManager": "YARN_NODEMANAGER_OPTS"}
YARN_OPTS = (
    '%s="-Dcom.sun.management.jmxremote.ssl=false -Dcom.sun.management.jmxremote.authenticate=false '
    '-Dcom.sun.management.jmxremote.port=%d $%s"'
)
YARN_ENV_PATH = "/usr/local/hadoop/etc/hadoop/yarn-env.sh"
HADOOPJMX_CONFIG = string.Template(
    """
monitors:
  - type: collectd/hadoopjmx
    host: $host
    port: $port
    nodeType: $nodeType
    customDimensions:
      nodeType: $nodeType
"""
)


@pytest.mark.flaky(reruns=2, reruns_delay=5)
@pytest.mark.parametrize("version", ["2.9.1", "3.0.3"])
@pytest.mark.parametrize("node_type", NODETYPE_PORT.keys())
def test_hadoopjmx(version, node_type):
    """
    Any new versions of hadoop should be manually built, tagged, and pushed to quay.io, i.e.
    docker build \
        -t quay.io/signalfx/hadoop-test:<version> \
        --build-arg HADOOP_VER=<version> \
        <repo_root>/test-services/hadoop
    docker push quay.io/signalfx/hadoop-test:<version>
    """
    with run_container("quay.io/signalfx/hadoop-test:%s" % version, hostname="hadoop-master") as hadoop_master:
        with run_container("quay.io/signalfx/hadoop-test:%s" % version, hostname="hadoop-worker1") as hadoop_worker1:
            if node_type in ["nameNode", "resourceManager"]:
                container = hadoop_master
            else:
                container = hadoop_worker1
            host = container_ip(container)
            port = NODETYPE_PORT[node_type]
            if node_type in ["resourceManager", "nodeManager"]:
                yarn_var = YARN_VAR[node_type]
                yarn_opts = YARN_OPTS % (yarn_var, port, yarn_var)
                cmd = ["/bin/bash", "-c", "echo 'export %s' >> %s" % (yarn_opts, YARN_ENV_PATH)]
                container.exec_run(cmd)

            start_hadoop(hadoop_master, hadoop_worker1)

            # wait for jmx to be available
            assert wait_for(p(tcp_socket_open, host, port), 60), "jmx service not listening on port %d" % port

            # start the agent with hadoopjmx config
            config = HADOOPJMX_CONFIG.substitute(host=host, port=port, nodeType=node_type)
            with Agent.run(config) as agent:
                assert wait_for(p(has_datapoint_with_dim, agent.fake_services, "nodeType", node_type)), (
                    "Didn't get hadoopjmx datapoints for nodeType %s" % node_type
                )

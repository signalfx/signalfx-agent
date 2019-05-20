from contextlib import contextmanager
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import tcp_socket_open, has_datapoint_with_dim
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_container, wait_for
from tests.helpers.verify import verify
from tests.monitors.collectd_hadoop.hadoop_test import start_hadoop

pytestmark = [pytest.mark.collectd, pytest.mark.hadoopjmx, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("collectd/hadoopjmx")
LATEST_VERSION = "3.0.3"
VERSIONS = ["2.9.1", LATEST_VERSION]
NODETYPE_PORT = {"nameNode": 5677, "dataNode": 5677, "resourceManager": 5680, "nodeManager": 8002}
NODETYPE_GROUP = {
    "nameNode": "name-node",
    "dataNode": "data-node",
    "resourceManager": "resource-manager",
    "nodeManager": "node-manager",
}
YARN_VAR = {"resourceManager": "YARN_RESOURCEMANAGER_OPTS", "nodeManager": "YARN_NODEMANAGER_OPTS"}
YARN_OPTS = (
    '%s="-Dcom.sun.management.jmxremote.ssl=false -Dcom.sun.management.jmxremote.authenticate=false '
    '-Dcom.sun.management.jmxremote.port=%d $%s"'
)
YARN_ENV_PATH = "/usr/local/hadoop/etc/hadoop/yarn-env.sh"
HADOOPJMX_CONFIG = """
monitors:
  - type: collectd/hadoopjmx
    host: {host}
    port: {port}
    nodeType: {nodeType}
    customDimensions:
      nodeType: {nodeType}
    extraMetrics: {extraMetrics}
"""


@contextmanager
def run_node(node_type, version):
    """
    Any new versions of hadoop should be manually built, tagged, and pushed to quay.io, i.e.
    docker build \
        -t quay.io/signalfx/hadoop-test:<version> \
        --build-arg HADOOP_VER=<version> \
        <repo_root>/test-services/hadoop
    docker push quay.io/signalfx/hadoop-test:<version>
    """
    with run_container(
        f"quay.io/signalfx/hadoop-test:{version}", hostname="hadoop-master"
    ) as hadoop_master, run_container(
        f"quay.io/signalfx/hadoop-test:{version}", hostname="hadoop-worker1"
    ) as hadoop_worker1:
        if node_type in ["nameNode", "resourceManager"]:
            container = hadoop_master
        else:
            container = hadoop_worker1
        host = container_ip(container)
        port = NODETYPE_PORT[node_type]
        if node_type in ["resourceManager", "nodeManager"]:
            yarn_var = YARN_VAR[node_type]
            yarn_opts = YARN_OPTS % (yarn_var, port, yarn_var)
            cmd = ["/bin/bash", "-c", f"echo 'export {yarn_opts}' >> {YARN_ENV_PATH}"]
            container.exec_run(cmd)

        start_hadoop(hadoop_master, hadoop_worker1)

        # wait for jmx to be available
        assert wait_for(p(tcp_socket_open, host, port)), f"JMX service not listening on port {port}"
        yield host, port


def run(version, node_type, metrics, extra_metrics=""):
    with run_node(node_type, version) as (host, port):
        # start the agent with hadoopjmx config
        config = HADOOPJMX_CONFIG.format(host=host, port=port, nodeType=node_type, extraMetrics=extra_metrics)
        with Agent.run(config) as agent:
            verify(agent, metrics)
            # Check for expected dimension.
            assert has_datapoint_with_dim(
                agent.fake_services, "nodeType", node_type
            ), f"Didn't get hadoopjmx datapoints for nodeType {node_type}"


@pytest.mark.flaky(reruns=2, reruns_delay=5)
@pytest.mark.parametrize("version", VERSIONS)
@pytest.mark.parametrize("node_type", NODETYPE_PORT.keys())
def test_hadoopjmx_included(version, node_type):
    included = (
        METADATA.metrics_by_group[NODETYPE_GROUP[node_type]] | METADATA.metrics_by_group["jvm"]
    ) & METADATA.included_metrics
    run(version, node_type, included)


@pytest.mark.flaky(reruns=2, reruns_delay=5)
@pytest.mark.parametrize("node_type", NODETYPE_PORT.keys())
def test_hadoopjmx_all(node_type):
    # Just test latest for all metrics.
    run(
        LATEST_VERSION,
        node_type,
        METADATA.metrics_by_group[NODETYPE_GROUP[node_type]] | METADATA.metrics_by_group["jvm"],
        extra_metrics="['*']",
    )

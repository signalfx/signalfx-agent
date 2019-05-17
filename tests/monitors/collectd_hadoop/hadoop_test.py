from functools import partial as p
from pathlib import Path

import pytest

from tests.helpers.assertions import has_datapoint, has_datapoint_with_dim, http_status, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, print_lines, run_container, wait_for
from tests.helpers.verify import run_agent_verify

pytestmark = [pytest.mark.collectd, pytest.mark.hadoop, pytest.mark.monitor_with_endpoints]

SCRIPT_DIR = Path(__file__).parent.resolve()
METADATA = Metadata.from_package("collectd/hadoop")

# TODO: figure out how to get more metrics
EXCLUDED = {
    "gauge.hadoop.cluster.metrics.total_mb",
    "gauge.hadoop.cluster.metrics.total_virtual_cores",
    "gauge.hadoop.mapreduce.job.elapsedTime",
    "gauge.hadoop.mapreduce.job.failedMapAttempts",
    "gauge.hadoop.mapreduce.job.failedReduceAttempts",
    "gauge.hadoop.mapreduce.job.mapsTotal",
    "gauge.hadoop.mapreduce.job.successfulMapAttempts",
    "gauge.hadoop.mapreduce.job.successfulReduceAttempts",
    "gauge.hadoop.resource.manager.apps.allocatedMB",
    "gauge.hadoop.resource.manager.apps.allocatedVCores",
    "gauge.hadoop.resource.manager.apps.clusterUsagePercentage",
    "gauge.hadoop.resource.manager.apps.memorySeconds",
    "gauge.hadoop.resource.manager.apps.priority",
    "gauge.hadoop.resource.manager.apps.progress",
    "gauge.hadoop.resource.manager.apps.queueUsagePercentage",
    "gauge.hadoop.resource.manager.apps.runningContainers",
    "gauge.hadoop.resource.manager.apps.vcoreSeconds",
}


def distribute_hostnames(containers):
    """
    iterate over each container and pass its hostname and ip to etc host on
    all of the other containers in the dictionary
    """
    for hostname, container in containers.items():
        ip_addr = container_ip(container)
        for target in containers:
            if hostname != target:
                containers[target].exec_run(
                    ["/bin/bash", "-c", "echo '{0} {1}' >> /etc/hosts".format(ip_addr, hostname)]
                )


def start_hadoop(hadoop_master, hadoop_worker1):
    containers = {"hadoop-master": hadoop_master, "hadoop-worker1": hadoop_worker1}

    # distribute the ip and hostnames for each container
    distribute_hostnames(containers)

    # format hdfs
    print_lines(hadoop_master.exec_run(["/usr/local/hadoop/bin/hdfs", "namenode", "-format"])[1])

    # start hadoop and yarn
    print_lines(hadoop_master.exec_run("start-dfs.sh")[1])
    print_lines(hadoop_master.exec_run("start-yarn.sh")[1])

    # wait for yarn api to be available
    host = container_ip(hadoop_master)
    assert wait_for(p(tcp_socket_open, host, 8088), 60), "service not listening on port"
    assert wait_for(p(http_status, url="http://{0}:8088".format(host), status=[200]), 120), "service didn't start"

    return host


@pytest.mark.flaky(reruns=2, reruns_delay=5)
@pytest.mark.parametrize("version", ["2.9.1", "3.0.3"])
def test_hadoop_included(version):
    """
    Any new versions of hadoop should be manually built, tagged, and pushed to quay.io, i.e.
    docker build \
        -t quay.io/signalfx/hadoop-test:<version> \
        --build-arg HADOOP_VER=<version> \
        <repo_root>/test-services/hadoop
    docker push quay.io/signalfx/hadoop-test:<version>
    """
    with run_container(
        "quay.io/signalfx/hadoop-test:%s" % version, hostname="hadoop-master"
    ) as hadoop_master, run_container(
        "quay.io/signalfx/hadoop-test:%s" % version, hostname="hadoop-worker1"
    ) as hadoop_worker1:
        host = start_hadoop(hadoop_master, hadoop_worker1)

        # start the agent with hadoop config
        config = f"""
            monitors:
              - type: collectd/hadoop
                host: {host}
                port: 8088
                verbose: true
            """
        agent = run_agent_verify(config, METADATA.included_metrics - EXCLUDED)
        assert has_datapoint_with_dim(agent.fake_services, "plugin", "apache_hadoop"), "Didn't get hadoop datapoints"
        assert has_datapoint(
            agent.fake_services, "gauge.hadoop.cluster.metrics.active_nodes", {}, 1
        ), "expected 1 hadoop worker node"

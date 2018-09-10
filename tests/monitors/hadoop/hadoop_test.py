from functools import partial as p
import os
import pytest
import string

from tests.helpers.util import wait_for, run_agent, run_service, run_container, container_ip, print_lines
from tests.helpers.assertions import tcp_socket_open, has_datapoint, has_datapoint_with_dim, http_status
from tests.helpers.util import (
    get_monitor_metrics_from_selfdescribe,
    get_monitor_dims_from_selfdescribe
)
from tests.kubernetes.utils import (
    run_k8s_monitors_test,
    get_discovery_rule,
)

pytestmark = [pytest.mark.collectd, pytest.mark.hadoop, pytest.mark.monitor_with_endpoints]

hadoop_config = string.Template("""
monitors:
  - type: collectd/hadoop
    host: $host
    port: $port
    verbose: true
""")


def distribute_hostnames(containers):
    """
    iterate over each container and pass its hostname and ip to etc host on
    all of the other containers in the dictionary
    """
    for hostname, container in containers.items():
        ip = container_ip(container)
        for target in containers:
            if hostname != target:
                containers[target].exec_run(["/bin/bash", "-c", "echo '{0} {1}' >> /etc/hosts".format(ip, hostname)])


@pytest.mark.parametrize("version", ["2.9.1"])#, "3.0.3"])
def test_hadoop(version):
    with run_service("hadoop", buildargs={"HADOOP_VER": version}, hostname="hadoop-master") as hadoop_master:
        with run_container(hadoop_master.image, hostname="hadoop-worker1") as hadoop_worker1:
            containers = {
                'hadoop-master': hadoop_master, 'hadoop-worker1': hadoop_worker1
            }

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

            # start the agent with hadoop config
            config = hadoop_config.substitute(host=host, port=8088)
            with run_agent(config) as [backend, _, _]:
                assert wait_for(p(has_datapoint_with_dim, backend, 'plugin', 'apache_hadoop')), "Didn't get hadoop datapoints"
                assert wait_for(p(has_datapoint, backend, 'gauge.hadoop.cluster.metrics.active_nodes', {}, 1)), "expected 1 hadoop worker node"

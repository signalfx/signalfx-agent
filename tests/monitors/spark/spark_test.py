"""
Tests for the collectd/spark monitor
"""
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for
from tests.helpers.verify import verify

pytestmark = [pytest.mark.collectd, pytest.mark.spark, pytest.mark.monitor_with_endpoints, pytest.mark.flaky(reruns=1)]

METADATA = Metadata.from_package("collectd/spark")

# TODO: figure out how to get these metrics
EXCLUDED = {
    "gauge.jvm.MarkSweepCompact.count",
    "gauge.jvm.MarkSweepCompact.time",
    "gauge.jvm.pools.Eden-Space.committed",
    "gauge.jvm.pools.Eden-Space.used",
    "gauge.jvm.pools.Survivor-Space.committed",
    "gauge.jvm.pools.Survivor-Space.used",
    "gauge.jvm.pools.Tenured-Gen.committed",
    "gauge.jvm.pools.Tenured-Gen.used",
}

SPARK_APP = "examples/src/main/python/streaming/network_wordcount.py localhost 9999"


def run(config, metrics):
    with run_service("spark", command="bin/spark-class org.apache.spark.deploy.master.Master") as spark_master:
        master_ip = container_ip(spark_master)
        assert wait_for(p(tcp_socket_open, master_ip, 7077), 60), "master service didn't start"
        assert wait_for(p(tcp_socket_open, master_ip, 8080), 60), "master webui service didn't start"
        assert spark_master.exec_run("./sbin/start-history-server.sh").exit_code == 0, "history service didn't start"

        with run_service(
            "spark", command=f"bin/spark-class org.apache.spark.deploy.worker.Worker spark://{master_ip}:7077"
        ) as spark_worker:
            worker_ip = container_ip(spark_worker)
            assert wait_for(p(tcp_socket_open, worker_ip, 8081), 60), "worker webui service didn't start"

            spark_master.exec_run("nc -lk 9999", detach=True)
            spark_master.exec_run(
                f"bin/spark-submit --master spark://{master_ip}:7077 --conf spark.driver.host={master_ip} {SPARK_APP}",
                detach=True,
            )
            assert wait_for(p(tcp_socket_open, master_ip, 4040), 60), "application service didn't start"

            config = config.format(master_ip=master_ip, worker_ip=worker_ip)
            with Agent.run(config) as agent:
                verify(agent, metrics, timeout=60)
                assert has_datapoint_with_dim(
                    agent.fake_services, "plugin", "apache_spark"
                ), "Didn't get spark datapoints"


def test_spark_default():
    run(
        """
        monitors:
        - type: collectd/spark
          host: {master_ip}
          port: 8080
          clusterType: Standalone
          isMaster: true
          collectApplicationMetrics: true
        - type: collectd/spark
          host: {worker_ip}
          port: 8081
          clusterType: Standalone
          isMaster: false
        """,
        METADATA.default_metrics,
    )


def test_spark_all():
    run(
        """
        monitors:
        - type: collectd/spark
          host: {master_ip}
          port: 8080
          clusterType: Standalone
          isMaster: true
          collectApplicationMetrics: true
          enhancedMetrics: true
        - type: collectd/spark
          host: {worker_ip}
          port: 8081
          clusterType: Standalone
          isMaster: false
          enhancedMetrics: true
        """,
        METADATA.all_metrics - EXCLUDED,
    )

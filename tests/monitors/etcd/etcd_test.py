from contextlib import contextmanager
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_container, wait_for, run_service
from tests.helpers.verify import run_agent_verify, verify_expected_is_subset
from tests.paths import TEST_SERVICES_DIR

pytestmark = [pytest.mark.collectd, pytest.mark.etcd, pytest.mark.monitor_with_endpoints]

ETCD_CONFIG = """
monitors:
  - type: collectd/etcd
    host: {host}
    port: 2379
    clusterName: test-cluster
"""

ETCD_TLS_CONFIG = """
monitors:
  - type: collectd/etcd
    host: {host}
    port: {port}
    clusterName: test-cluster
    sslCertificate: {testServices}/etcd/certs/client.crt
    sslKeyFile: {testServices}/etcd/certs/client.key
    sslCACerts: {testServices}/etcd/certs/server.crt
    skipSSLValidation: {skipValidation}
"""

METADATA = Metadata.from_package("collectd/etcd")
EXCLUDED_METRICS = {
    # leader metrics don't occur on this test environment
    "counter.etcd.leader.counts.success",
    "gauge.etcd.leader.latency.current",
    "counter.etcd.leader.counts.fail",
    "gauge.etcd.leader.latency.max",
    "gauge.etcd.leader.latency.average",
    "gauge.etcd.leader.latency.stddev",
    "gauge.etcd.leader.latency.min",
}
INCLUDED_METRICS = METADATA.included_metrics - EXCLUDED_METRICS
ENHANCED_METRICS = METADATA.all_metrics - EXCLUDED_METRICS


@contextmanager
def run_etcd(tls=False, **kwargs):
    if tls:
        cmd = """
            --listen-client-urls https://0.0.0.0:2379
            --advertise-client-urls https://0.0.0.0:2379
            --trusted-ca-file /opt/testing/certs/server.crt
            --cert-file /opt/testing/certs/server.crt
            --key-file /opt/testing/certs/server.key
            --client-cert-auth
        """
        # NOTE: If running in a container this will only work if the container is running in host
        # networking mode. We need to be able to connect to "localhost" since it is the CN in the
        # certificate.
        with run_service("etcd", command=cmd, **kwargs) as container:
            host = container_ip(container)
            assert wait_for(p(tcp_socket_open, host, 2379), 60), "service didn't start"
            yield container
    else:
        cmd = """
            --listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001
            --advertise-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001
        """
        with run_container("quay.io/coreos/etcd:v2.3.8", command=cmd) as container:
            host = container_ip(container)
            assert wait_for(p(tcp_socket_open, host, 2379), 60), "service didn't start"
            yield container


@pytest.mark.requires_host_networking
def test_etcd_tls_skip_validation():
    with run_etcd(tls=True) as container:
        host = container_ip(container)
        config = ETCD_TLS_CONFIG.format(host=host, port=2379, skipValidation="true", testServices=TEST_SERVICES_DIR)
        run_agent_verify(config, INCLUDED_METRICS)


@pytest.mark.requires_host_networking
def test_etcd_tls_validate():
    with run_etcd(tls=True, ports={"2379/tcp": None}) as container:
        host = "localhost"
        port = int(container.attrs["NetworkSettings"]["Ports"]["2379/tcp"][0]["HostPort"])
        config = ETCD_TLS_CONFIG.format(host=host, port=port, skipValidation="false", testServices=TEST_SERVICES_DIR)
        run_agent_verify(config, INCLUDED_METRICS)


def test_etcd_monitor():
    with run_etcd() as etcd_cont:
        host = container_ip(etcd_cont)
        config = ETCD_CONFIG.format(host=host)

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "etcd")
            ), "Didn't get etcd datapoints"


def test_etcd_monitor_included():
    with run_etcd() as etcd_cont:
        host = container_ip(etcd_cont)
        config = ETCD_CONFIG.format(host=host)
        run_agent_verify(config, INCLUDED_METRICS)


def test_etcd_monitor_enhanced():
    with run_etcd() as etcd_cont:
        host = container_ip(etcd_cont)

        with Agent.run(
            f"""
            monitors:
            - type: collectd/etcd
              host: {host}
              port: 2379
              clusterName: test-cluster
              enhancedMetrics: true
            """
        ) as agent:
            verify_expected_is_subset(agent, ENHANCED_METRICS)

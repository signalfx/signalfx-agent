from contextlib import contextmanager
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_container, wait_for, run_service
from tests.paths import TEST_SERVICES_DIR

pytestmark = [pytest.mark.collectd, pytest.mark.etcd, pytest.mark.monitor_with_endpoints]

ETCD_CONFIG = """
monitors:
  - type: etcd
    host: {host}
    port: 2379
    sendAllMetrics: true
"""

ETCD_TLS_CONFIG = """
monitors:
  - type: etcd
    host: {host}
    port: {port}
    useHTTPS: true
    clientCertPath: {testServices}/etcd/certs/client.crt
    clientKeyPath: {testServices}/etcd/certs/client.key
    caCertPath: {testServices}/etcd/certs/server.crt
    skipVerify: {skipValidation}
    sendAllMetrics: true
"""

METADATA = Metadata.from_package("etcd")


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


def test_etcd_tls_skip_validation():
    with run_etcd(tls=True) as container:
        host = container_ip(container)
        config = ETCD_TLS_CONFIG.format(host=host, port=2379, skipValidation="true", testServices=TEST_SERVICES_DIR)

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint, agent.fake_services, metric_name="process_cpu_seconds_total")
            ), "Didn't get etcd datapoints"


def test_etcd_monitor():
    with run_etcd() as etcd_cont:
        host = container_ip(etcd_cont)
        config = ETCD_CONFIG.format(host=host)

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint, agent.fake_services, metric_name="process_cpu_seconds_total")
            ), "Didn't get etcd datapoints"

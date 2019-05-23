import pytest

from tests.helpers.metadata import Metadata
from tests.helpers.util import run_container, container_ip
from tests.helpers.verify import run_agent_verify_default_metrics, run_agent_verify_all_metrics

VERSIONS = ["memcached:1.5-alpine", "memcached:latest"]
METADATA = Metadata.from_package("collectd/memcached")


@pytest.mark.parametrize("version", VERSIONS)
def test_memcached_default(version):
    with run_container(version) as container:
        host = container_ip(container)
        run_agent_verify_default_metrics(
            f"""
            monitors:
            - type: collectd/memcached
              host: {host}
              port: 11211
            """,
            METADATA,
        )


@pytest.mark.parametrize("version", VERSIONS)
def test_memcached_all(version):
    with run_container(version) as container:
        host = container_ip(container)
        run_agent_verify_all_metrics(
            f"""
            monitors:
            - type: collectd/memcached
              host: {host}
              port: 11211
              extraMetrics: ["*"]
            """,
            METADATA,
        )

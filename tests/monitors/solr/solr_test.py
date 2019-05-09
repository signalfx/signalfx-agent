import time
from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, run_service, wait_for
from tests.helpers.verify import verify_expected_is_superset

pytestmark = [pytest.mark.collectd, pytest.mark.etcd, pytest.mark.monitor_with_endpoints]

METADATA = Metadata.from_package("collectd/solr")


def test_solr_monitor_defaults():
    with run_service("solr") as solr_container:
        host = container_ip(solr_container)
        config = dedent(
            f"""
        monitors:
        - type: collectd/solr
          host: {host}
          port: 8983
        """
        )
        assert wait_for(p(tcp_socket_open, host, 8983), 60), "service not listening on port"
        with Agent.run(config) as agent:
            time.sleep(10)
            assert agent.fake_services.datapoints
            # We don't get all default metrics but this ensures we don't get
            # any non-default metrics through with default config.
            verify_expected_is_superset(agent, METADATA.included_metrics)

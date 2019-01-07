import string
from functools import partial as p

import pytest

from helpers.assertions import has_datapoint_with_dim, has_log_message, tcp_socket_open, has_datapoint_with_metric_name
from helpers.util import container_ip, run_service, run_agent, wait_for, ensure_always

pytestmark = [pytest.mark.collectd, pytest.mark.etcd, pytest.mark.monitor_with_endpoints]


def test_solr_monitor():
    monitor_config = string.Template(
        """
    monitors:
    - type: collectd/solr
      host: $host
      port: 8983
    """
    )
    with run_service("solr") as solr_container:
        host = container_ip(solr_container)
        config = monitor_config.substitute(host=host)
        assert wait_for(p(tcp_socket_open, host, 8983), 60), "service not listening on port"

        with run_agent(config) as [backend, get_output, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "solr")), "Didn't get solr datapoints"
            assert ensure_always(lambda: has_datapoint_with_metric_name(backend, "counter.solr.http_5xx_responses"))
            assert not has_log_message(get_output().lower(), "error"), "error found in agent output!"

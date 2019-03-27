"""
Tests for the expvar monitor
"""
from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.util import container_ip, get_monitor_metrics_from_metadata_yaml, run_agent, run_service, wait_for

pytestmark = [pytest.mark.expvar, pytest.mark.monitor_with_endpoints]


def test_nginx():
    with run_service("expvar") as expvar_container:
        host = container_ip(expvar_container)
        assert wait_for(p(tcp_socket_open, host, 8080), 60), "service didn't start"

        with run_agent(
            dedent(
                f"""
          monitors:
           - type: expvar
             host: {host}
             port: 8080
         """
            )
        ) as [backend, _, _]:
            metrics_defined = get_monitor_metrics_from_metadata_yaml("internal/monitors/expvar")
            for metric in metrics_defined:
                if metric.get("included", False):
                    print("Waiting for %s" % metric)
                    assert wait_for(
                        p(has_datapoint, backend, metric_name=metric["name"])
                    ), "Didn't get included datapoints"

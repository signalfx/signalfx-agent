from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import any_metric_found
from tests.helpers.util import container_ip, get_monitor_metrics_from_selfdescribe, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.openstack, pytest.mark.monitor_without_endpoints]


@pytest.mark.flaky(reruns=1)
def test_openstack(devstack):
    host = container_ip(devstack)
    config = dedent(
        f"""
            monitors:
              - type: collectd/openstack
                authURL: http://{host}/identity/v3
                username: admin
                password: testing123
        """
    )
    with Agent.run(config) as agent:
        expected_metrics = get_monitor_metrics_from_selfdescribe("collectd/openstack")
        assert wait_for(
            p(any_metric_found, agent.fake_services, expected_metrics), 60
        ), "Timed out waiting for openstack metrics"

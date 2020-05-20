import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint
from tests.helpers.metadata import Metadata
from tests.helpers.util import run_service
from tests.helpers.verify import verify

pytestmark = [pytest.mark.monitor_without_endpoints]


METADATA = Metadata.from_package("cgroups")


def test_cgroup_monitor():
    with run_service(
        "nginx", cpu_period=100_000, cpu_quota=10000, cpu_shares=50, mem_limit=20 * 1024 * 1024
    ) as nginx_container:
        with Agent.run(
            """
    monitors:
      - type: cgroups
        extraMetrics: ['*']
    """
        ) as agent:
            verify(agent, METADATA.all_metrics)

            expected_cgroup = "/docker/" + nginx_container.id

            assert has_datapoint(
                agent.fake_services, metric_name="cgroup.cpu_shares", value=50, dimensions={"cgroup": expected_cgroup}
            )

            assert has_datapoint(
                agent.fake_services,
                metric_name="cgroup.cpu_cfs_period_us",
                value=100_000,
                dimensions={"cgroup": expected_cgroup},
            )

            assert has_datapoint(
                agent.fake_services,
                metric_name="cgroup.cpu_cfs_quota_us",
                value=10000,
                dimensions={"cgroup": expected_cgroup},
            )

            assert has_datapoint(
                agent.fake_services,
                metric_name="cgroup.memory_limit_in_bytes",
                value=20 * 1024 * 1024,
                dimensions={"cgroup": expected_cgroup},
            )

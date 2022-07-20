from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint
from tests.helpers.metadata import Metadata
from tests.helpers.util import run_service, wait_for
from tests.helpers.verify import verify_expected_is_subset

pytestmark = [pytest.mark.monitor_without_endpoints]


METADATA = Metadata.from_package("cgroups")


def test_cgroup_monitor():
    with run_service(
        "nginx", cpu_period=100_000, cpu_quota=10000, cpu_shares=50, mem_limit=20 * 1024 * 1024, cgroup_parent="/docker"
    ) as nginx_container:
        with Agent.run(
            """
    monitors:
      - type: cgroups
        extraMetrics: ['*']
    """
        ) as agent:
            expected = METADATA.all_metrics - {
                # these aren't reliably reported by all docker runtimes
                "cgroup.memory_stat_hierarchical_memsw_limit",
                "cgroup.memory_stat_swap",
                "cgroup.memory_stat_total_swap",
            }
            verify_expected_is_subset(agent, expected)

            expected_cgroup = "/docker/" + nginx_container.id

            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="cgroup.cpu_shares",
                    value=50,
                    dimensions={"cgroup": expected_cgroup},
                )
            )

            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="cgroup.cpu_cfs_period_us",
                    value=100_000,
                    dimensions={"cgroup": expected_cgroup},
                )
            )

            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="cgroup.cpu_cfs_quota_us",
                    value=10000,
                    dimensions={"cgroup": expected_cgroup},
                )
            )

            assert wait_for(
                p(
                    has_datapoint,
                    agent.fake_services,
                    metric_name="cgroup.memory_limit_in_bytes",
                    value=20 * 1024 * 1024,
                    dimensions={"cgroup": expected_cgroup},
                )
            )

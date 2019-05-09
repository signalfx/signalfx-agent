from tests.helpers.agent import Agent
from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify
from tests.monitors.expvar.expvar_test import run_expvar


def test_extra_metrics_passthrough():
    """
    The specified extraMetrics should be allowed through even though they are
    not included by default.
    """
    metadata = Metadata.from_package("expvar")

    with run_expvar() as expvar_container_ip:
        with Agent.run(
            f"""
               monitors:
                 - type: expvar
                   host: {expvar_container_ip}
                   port: 8080
                   intervalSeconds: 1
                   extraMetrics:
                    - memstats.by_size.mallocs
               """
        ) as agent:
            assert "memstats.by_size.mallocs" in metadata.nonincluded_metrics
            verify(agent, metadata.included_metrics | {"memstats.by_size.mallocs"})

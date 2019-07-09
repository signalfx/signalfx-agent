from functools import partial as p

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import container_ip, ensure_always, run_service, wait_for


def test_new_monitor_filtering():
    with Agent.run(
        """
           monitors:
             - type: internal-metrics
               intervalSeconds: 1
               datapointsToExclude:
                - metricNames:
                  - '*'
                  - '!sfxagent.go_heap_*'
                  - '!sfxagent.go_frees'
           """
    ) as agent:
        is_expected = lambda dp: dp.metric.startswith("sfxagent.go_heap") or dp.metric == "sfxagent.go_frees"

        def no_filtered_metrics():
            for dp in agent.fake_services.datapoints:
                assert is_expected(dp), f"Got unexpected metric name {dp.metric}"
            return True

        assert wait_for(lambda: agent.fake_services.datapoints), "No datapoints received"
        assert ensure_always(no_filtered_metrics, interval_seconds=2, timeout_seconds=5)

        metrics_received = agent.fake_services.datapoints_by_metric.keys()
        assert "sfxagent.go_frees" in metrics_received
        assert "sfxagent.go_heap_inuse" in metrics_received
        assert "sfxagent.go_heap_released" in metrics_received


def test_filtering_by_dimensions():
    with run_service("nginx") as nginx_cont:
        with Agent.run(
            f"""
               monitors:
                 - type: collectd/nginx
                   host: {container_ip(nginx_cont)}
                   port: 80
                   intervalSeconds: 1
                   datapointsToExclude:
                    - dimensions:
                        plugin: ['*', '!nginx']
               """
        ) as agent:
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"plugin": "nginx"})
            ), "Didn't get nginx datapoints"

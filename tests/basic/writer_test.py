import time
from textwrap import dedent

from tests.helpers.agent import Agent
from tests.helpers.util import container_ip, run_service

BASIC_CONFIG = """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/cpu
  - type: collectd/uptime
"""


def test_writer_no_skipped_datapoints():
    """
    See if we get every datapoint that we expect
    """
    num_metrics = 1000
    with run_service("dpgen", environment={"NUM_METRICS": num_metrics}) as dpgen_cont:
        with Agent.run(
            dedent(
                f"""
             writer:
               maxRequests: 1
               datapointMaxBatchSize: 100
               maxDatapointsBuffered: 1947
             monitors:
             - type: prometheus-exporter
               host: {container_ip(dpgen_cont)}
               port: 3000
               intervalSeconds: 1
        """
            )
        ) as agent:
            time.sleep(10)
            dpgen_cont.remove(force=True, v=True)
            time.sleep(2)

            assert agent.fake_services.datapoints, "Didn't get any datapoints"
            assert len(agent.fake_services.datapoints) % num_metrics == 0, "Didn't get 1000n datapoints"
            for i in range(0, num_metrics):
                assert (
                    len(
                        [
                            dp
                            for dp in agent.fake_services.datapoints
                            if [dim for dim in dp.dimensions if dim.key == "index" and dim.value == str(i)]
                        ]
                    )
                    == len(agent.fake_services.datapoints) / num_metrics
                ), "Didn't get each datapoint n times"

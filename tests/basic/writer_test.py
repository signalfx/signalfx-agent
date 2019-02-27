import time
from textwrap import dedent

from tests.helpers.util import container_ip, run_agent, run_service

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
        with run_agent(
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
        ) as [backend, _, _]:
            time.sleep(10)
            dpgen_cont.remove(force=True, v=True)
            time.sleep(2)

            assert backend.datapoints, "Didn't get any datapoints"
            assert len(backend.datapoints) % num_metrics == 0, "Didn't get 1000n datapoints"
            for i in range(0, num_metrics):
                assert (
                    len(
                        [
                            dp
                            for dp in backend.datapoints
                            if [dim for dim in dp.dimensions if dim.key == "index" and dim.value == str(i)]
                        ]
                    )
                    == len(backend.datapoints) / num_metrics
                ), "Didn't get each datapoint n times"

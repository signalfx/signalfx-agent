import time
from textwrap import dedent

from tests.helpers.agent import Agent
from tests.helpers.util import container_ip, run_service


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


def test_splunk_output():
    """
    See if we Splunk output for datapoints
    """
    num_metrics = 1000
    with run_service("dpgen", environment={"NUM_METRICS": num_metrics}) as dpgen_cont:
        with Agent.run(
            dedent(
                f"""
             signalFxRealm: null

             writer:
               splunk:
                 enabled: true

             monitors:
             - type: prometheus-exporter
               host: {container_ip(dpgen_cont)}
               port: 3000
               intervalSeconds: 1
        """
            ),
            backend_options={"splunk_hec_port": 0},
        ) as agent:
            time.sleep(10)
            dpgen_cont.remove(force=True, v=True)
            time.sleep(2)

            assert agent.fake_services.splunk_entries, "Didn't get any splunk entries"
            assert len(agent.fake_services.splunk_entries) % num_metrics == 0, "Didn't get 1000n entries"
            for i in range(0, num_metrics):
                assert (
                    len(
                        [
                            e
                            for e in agent.fake_services.splunk_entries
                            if {k: v for k, v in e.get("fields", {}).items() if k == "index" and v == str(i)}
                        ]
                    )
                    == len(agent.fake_services.splunk_entries) / num_metrics
                ), "Didn't get each splunk entry n times"


def test_splunk_event_output():
    """
    See if we Splunk output for events
    """
    with Agent.run(
        dedent(
            """
         signalFxRealm: null

         writer:
           splunk:
             enabled: true

         monitors:
         - type: processlist
    """
        ),
        backend_options={"splunk_hec_port": 0},
    ) as agent:
        time.sleep(10)

        assert agent.fake_services.splunk_entries, "Didn't get any splunk entries"
        assert agent.fake_services.splunk_entries[0]["event"]["eventType"] == "objects.top-info"

from functools import partial as p

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import wait_for
from tests.monitors.kafka.kafka_test import run_kafka


def test_endpoint_config_mapping():
    with Agent.run(
        """
      observers:
        - type: docker
      monitors:
        - type: collectd/kafka
          discoveryRule: container_name == "kafka-discovery" && port == 7099
          configEndpointMappings:
            extraDimensions: 'Get(container_labels, "com.signalfx.extraDimensions")'
            clusterName: 'Get(container_labels, "com.signalfx.cluster")'
      """
    ) as agent:
        with run_kafka(
            "1.1.1",
            name="kafka-discovery",
            labels={"com.signalfx.extraDimensions": "{a: 1}", "com.signalfx.cluster": "prod"},
        ):
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"a": "1", "cluster": "prod"})
            ), "Didn't get kafka datapoints with properly mapped config"


def test_extra_dimension_endpoint_mapping():
    with Agent.run(
        """
      observers:
        - type: docker
      monitors:
        - type: collectd/kafka
          discoveryRule: container_name == "kafka-dimension-maps" && port == 7099
          clusterName: test
          extraDimensions:
            a: "1"
          extraDimensionsFromEndpoint:
            env: 'Get(container_labels, "app.signalfx.com/env")'
      """
    ) as agent:
        with run_kafka("1.1.1", name="kafka-dimension-maps", labels={"app.signalfx.com/env": "prod"}):
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"a": "1", "env": "prod"})
            ), "Didn't get kafka datapoints with properly mapped dimensions"

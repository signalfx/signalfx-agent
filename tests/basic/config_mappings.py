from functools import partial as p

from tests.helpers.assertions import has_datapoint
from tests.helpers.util import run_agent, wait_for
from tests.monitors.kafka.kafka_test import run_kafka

CONFIG = """
observers:
  - type: docker
monitors:
  - type: collectd/kafka
    discoveryRule: container_name == "kafka-discovery" && port == 7099
    configEndpointMappings:
      extraDimensions: 'Get(container_labels, "com.signalfx.extraDimensions")'
      clusterName: 'Get(container_labels, "com.signalfx.cluster")'

"""


def test_endpoint_config_mapping():
    with run_agent(CONFIG) as [backend, _, _]:
        with run_kafka(
            "1.1.1",
            name="kafka-discovery",
            labels={"com.signalfx.extraDimensions": "{a: 1}", "com.signalfx.cluster": "prod"},
        ):
            assert wait_for(
                p(has_datapoint, backend, dimensions={"a": "1", "cluster": "prod"})
            ), "Didn't get kafka datapoints with properly mapped config"

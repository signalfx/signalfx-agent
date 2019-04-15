import re
import time
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, has_datapoint_with_metric_name, regex_search_matches_output
from tests.helpers.util import send_udp_message, wait_for

pytestmark = [pytest.mark.windows, pytest.mark.telegraf_statsd, pytest.mark.telegraf]

# regex used to scrape the address and port that dogstatsd is listening on
STATSD_RE = re.compile(r"(?<=listener listening on:(?=(\s+)(?=(\d+.\d+.\d+.\d+)\:(\d+))))")

MONITOR_CONFIG = """
monitors:
  - type: telegraf/statsd
    protocol: udp
    serviceAddress: "127.0.0.1:0"
    parseDataDogTags: true
    metricSeparator: '.'
"""


def test_telegraf_statsd():
    with Agent.run(MONITOR_CONFIG) as agent:
        # wait until the statsd plugin logs the address and port it is listening on
        assert wait_for(p(regex_search_matches_output, agent.get_output, STATSD_RE.search))

        # scrape the host and port that the statsd plugin is listening on
        regex_results = STATSD_RE.search(agent.output)

        host = regex_results.groups()[1]
        port = int(regex_results.groups()[2])

        # send datapoints to the statsd listener
        for _ in range(0, 10):
            send_udp_message(host, port, "statsd.test.metric:55555|g|#dimension1:value1,dimension2:value2")
            time.sleep(1)

        # wait for fake ingest to receive the statsd metrics
        assert wait_for(p(has_datapoint_with_metric_name, agent.fake_services, "statsd.test.metric"))
        assert wait_for(
            p(has_datapoint_with_dim, agent.fake_services, "dimension1", "value1")
        ), "datapoint didn't have datadog tag"

        # send datapoints to the statsd listener
        for _ in range(0, 10):
            send_udp_message(host, port, "dogstatsd.test.metric:55555|g|#dimension1:,dimension2:value2")
            time.sleep(1)

        assert wait_for(
            p(has_datapoint_with_metric_name, agent.fake_services, "dogstatsd.test.metric")
        ), "didn't report metric with valueless datadog tag"

from functools import partial as p
import pytest
import time

from tests.helpers.util import *
from tests.helpers.assertions import *


pytestmark = [pytest.mark.windows,
              pytest.mark.telegraf_statsd,
              pytest.mark.telegraf]

# regex used to scrape the address and port that dogstatsd is listening on
statsdRE = re.compile(r'(?<=listener listening on:(?=(\s+)(?=(\d+.\d+.\d+.\d+)\:(\d+))))')

monitor_config = """
monitors:
  - type: telegraf/statsd
    protocol: udp
    serviceAddress: "127.0.0.1:0"
    parseDataDogTags: true
    metricSeparator: '.'
"""


def test_telegraf_statsd():
    with run_agent(monitor_config) as [backend, get_output, _]:
        # wait until the statsd plugin logs the address and port it is listening on
        assert wait_for(p(regex_search_matches_output, get_output, statsdRE.search))

        # scrape the host and port that the statsd plugin is listening on
        regex_results = statsdRE.search(get_output())

        host = regex_results.groups()[1]
        port = int(regex_results.groups()[2])

        # send datapoints to the statsd listener
        for i in range(0,10):
            send_udp_message(host, port, "statsd.test.metric:55555|g|#dimension1:value1,dimension2:value2")
            time.sleep(1)

        # wait for fake ingest to receive the statsd metrics
        assert wait_for(p(has_datapoint_with_metric_name, backend, "statsd.test.metric"))

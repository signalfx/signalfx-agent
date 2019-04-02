import re
import string
import time
from functools import partial as p

import pytest
from tests.helpers.assertions import has_datapoint_with_metric_name, regex_search_matches_output, udp_port_open_locally
from tests.helpers.util import fake_backend, run_agent_with_fake_backend, send_udp_message, wait_for

pytestmark = [pytest.mark.collectd, pytest.mark.dogstatsd, pytest.mark.monitor_without_endpoints]

# regex used to scrape the address and port taht dogstatsd is listening on
DOGSTATSD_RE = re.compile(r"(?<=dogstatsd:Listening on host & port: \(\')((\d.\d.\d.\d)\',\s(\d+))")

DOGSTATSD_CONFIG = string.Template(
    """
monitors:
- type: collectd/cpu
- type: collectd/signalfx-metadata
  verbose: true
  token: "RANDOMTOKEN"
  dogStatsDPort: 0
  ingestEndpoint: $ingestEndpoint
"""
)


def test_collectd_dogstatsd():
    with fake_backend.start() as backend:
        # configure the dogstatsd plugin to send to fake ingest
        config = DOGSTATSD_CONFIG.substitute(ingestEndpoint=backend.ingest_url)

        # start the agent with the dogstatsd plugin config
        with run_agent_with_fake_backend(config, backend) as [get_output, _, _, _]:
            # wait until the dogstatsd plugin logs the address and port it is listening on
            assert wait_for(p(regex_search_matches_output, get_output, DOGSTATSD_RE.search))

            # scrape the host and port that the dogstatsd plugin is listening on
            regex_results = DOGSTATSD_RE.search(get_output())
            host = regex_results.groups()[1]
            port = int(regex_results.groups()[2])

            # wait for dogstatsd port to open
            assert wait_for(p(udp_port_open_locally, port))

            # send datapoints to the dogstatsd listener
            for _ in range(0, 10):
                send_udp_message(host, port, "dogstatsd.test.metric:55555|g|#dimension1:value1,dimension2:value2")
                time.sleep(1)

            # wait for fake ingest to receive the dogstatsd metrics
            assert wait_for(p(has_datapoint_with_metric_name, backend, "dogstatsd.test.metric"))

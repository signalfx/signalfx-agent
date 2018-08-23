from functools import partial as p
import pytest
import string
import time

from tests.helpers.util import wait_for, run_agent, run_agent_with_fake_backend, run_service, container_ip, send_udp_message, fake_backend
from tests.helpers.assertions import *

pytestmark = [pytest.mark.collectd, pytest.mark.dogstatsd, pytest.mark.monitor_without_endpoints]

# regex used to scrape the address and port taht dogstatsd is listening on
dogstatsdRE = re.compile(r'(?<=dogstatsd:Listening on host & port: \(\')((\d.\d.\d.\d)\',\s(\d+))')

dogstatsd_config = string.Template("""
monitors:
- type: collectd/cpu
- type: collectd/signalfx-metadata
  verbose: true
  token: "RANDOMTOKEN"
  dogStatsDPort: 0
  ingestEndpoint: $ingestEndpoint
""")

def test_collectd_dogstatsd():
    with fake_backend.start() as f_backend:
        # configure the dogstatsd plugin to send to fake ingest
        config = dogstatsd_config.substitute(ingestEndpoint=f_backend.ingest_url)

        # start the agent with the dogstatsd plugin config
        with run_agent_with_fake_backend(config, f_backend) as [backend, get_output, _]:
            # wait until the dogstatsd plugin logs the address and port it is listening on
            assert wait_for(p(regex_search_matches_output, get_output, dogstatsdRE.search))

            # scrape the host and port that the dogstatsd plugin is listening on
            regex_results = dogstatsdRE.search(get_output())
            host = regex_results.groups()[1]
            port = int(regex_results.groups()[2])

            # wait for dogstatsd port to open
            assert wait_for(p(udp_port_open_locally, port))

            # send datapoints to the dogstatsd listener
            for i in range(0,10):
                send_udp_message(host, port, "dogstatsd.test.metric:55555|g|#dimension1:value1,dimension2:value2")
                time.sleep(1)

            # wait for fake ingest to receive the dogstatsd metrics
            assert wait_for(p(has_datapoint_with_metric_name, backend, "dogstatsd.test.metric"))

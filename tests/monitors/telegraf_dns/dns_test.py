from functools import partial as p
from textwrap import dedent
import pytest
from tests.helpers.metadata import Metadata
from signalfx.generated_protocol_buffers import signal_fx_protocol_buffers_pb2 as sf_pbuf
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, has_datapoint_with_dim
from tests.helpers.verify import run_agent_verify_default_metrics
from tests.helpers.util import wait_for

pytestmark = [pytest.mark.telegraf]

METADATA = Metadata.from_package("telegraf/monitors/dns")
SERVER = "1.1.1.1"
DOMAIN = "signalfx.com"


def test_telegraf_dns_metrics():
    # Config to get every possible metrics
    agent_config = dedent(
        f"""
        monitors:
        - type: telegraf/dns
          servers:
            - {SERVER}
        """
    )
    run_agent_verify_default_metrics(agent_config, METADATA)


def test_telegraf_resolve():
    with Agent.run(
        f"""
        monitors:
        - type: telegraf/dns
          servers:
            - {SERVER}
          domains:
            - {DOMAIN}
          recordType: A
        """
    ) as agent:
        assert wait_for(p(has_datapoint, agent.fake_services, "dns.result_code", metric_type=sf_pbuf.GAUGE, value=0))
        assert wait_for(p(has_datapoint_with_dim, agent.fake_services, "server", SERVER))
        assert wait_for(p(has_datapoint_with_dim, agent.fake_services, "domain", DOMAIN))

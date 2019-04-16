import string
from functools import partial as p

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.util import container_ip, run_container, wait_for

pytestmark = [pytest.mark.telegraf, pytest.mark.snmp, pytest.mark.monitor_with_endpoints]

MONITOR_CONFIG = string.Template(
    """
monitors:
- type: telegraf/snmp
  agents:
    - "$host:161"
  version: 2
  community: "public"
  fields:
    - name: "uptime"
      oid: ".1.3.6.1.2.1.1.3.0"
"""
)


def test_snmp():
    # snmp-simulator source is available at: https://github.com/xeemetric/snmp-simulator
    # the hard coded uptime OID used in this test was fetched using the following command
    # $ snmpget -On -v2c -c public <snmp simulator host>:161 system.sysUpTime.0
    with run_container("xeemetric/snmp-simulator") as test_container:
        host = container_ip(test_container)
        config = MONITOR_CONFIG.substitute(host=host)

        with Agent.run(config) as agent:
            assert wait_for(
                p(has_datapoint_with_dim, agent.fake_services, "plugin", "telegraf-snmp")
            ), "didn't get database io datapoints"

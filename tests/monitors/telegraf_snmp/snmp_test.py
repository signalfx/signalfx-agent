from functools import partial as p
import pytest
import string

from helpers.util import wait_for, run_agent, run_container, container_ip
from helpers.assertions import has_datapoint_with_dim

pytestmark = [pytest.mark.telegraf, pytest.mark.snmp, pytest.mark.monitor_with_endpoints]

monitor_config = string.Template(
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
        config = monitor_config.substitute(host=host)

        with run_agent(config) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "telegraf-snmp")), "didn't get database io datapoints"

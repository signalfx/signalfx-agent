from functools import partial as p
from textwrap import dedent
import pytest
import sys

from helpers.util import wait_for, run_agent
from helpers.assertions import *

pytestmark = [
    pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows"),
    pytest.mark.windows,
    pytest.mark.utilization,
]


def test_utilization():
    config = dedent(
        """
        monitors:
         - type: system-utilization
        """
    )
    with run_agent(config) as [backend, get_output, _]:
        assert wait_for(p(has_datapoint, backend, metric_name="memory.utilization")), "memory.utilization missing"
        assert wait_for(p(has_datapoint, backend, metric_name="memory.free")), "memory.free missing"
        assert wait_for(p(has_datapoint, backend, metric_name="memory.used")), "memory.used missing"
        assert wait_for(p(has_datapoint, backend, metric_name="disk.utilization")), "disk.utilization missing"
        assert wait_for(p(has_datapoint, backend, metric_name="df_complex.free")), "df_complex.free missing"
        assert wait_for(p(has_datapoint, backend, metric_name="df_complex.used")), "df_complex.used missing"
        assert wait_for(p(has_datapoint, backend, metric_name="cpu.utilization")), "cpu.utilization missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="cpu.utilization_per_core")
        ), "cpu.utilization_per_core missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="network_interface.bytes_received.per_second")
        ), "network_interface.bytes_received.per_second missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="network_interface.bytes_sent.per_second")
        ), "network_interface.bytes_sent.per_second missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="network_interface.errors_received.per_second")
        ), "network_interface.errors_received.per_second missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="network_interface.errors_sent.per_second")
        ), "network_interface.errors_sent.per_second missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="disk.read_ops.per_second")
        ), "disk.read.per_second missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="disk.write_ops.per_second")
        ), "disk.write.per_second missing"
        assert wait_for(p(has_datapoint, backend, metric_name="paging_file.pct_usage")), "paging_file.pct_usage missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="vmpage.swapped_in.per_second")
        ), "vmpage.swapped_in.per_second missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="vmpage.swapped_out.per_second")
        ), "vmpage.swapped_out.per_second missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="vmpage.swapped.per_second")
        ), "vmpage.swapped.per_second missing"

from functools import partial as p
from textwrap import dedent
import pytest
import sys

from helpers.util import wait_for, run_agent
from helpers.assertions import *

pytestmark = [
    pytest.mark.skipif(sys.platform != "win32", reason="only runs on windows"),
    pytest.mark.windows,
    pytest.mark.dotnet,
]


def test_dotnet():
    config = dedent(
        """
        monitors:
         - type: dotnet
        """
    )
    with run_agent(config) as [backend, get_output, _]:
        assert wait_for(
            p(has_datapoint, backend, metric_name="net_clr_exceptions.num_exceps_thrown_sec")
        ), "net_clr_exceptions.num_exceps_thrown_sec missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="net_clr_locksandthreads.num_of_current_logical_threads")
        ), "net_clr_locksandthreads.num_of_current_logical_threads missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="net_clr_locksandthreads.num_of_current_physical_threads")
        ), "net_clr_locksandthreads.num_of_current_physical_threads missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="net_clr_locksandthreads.contention_rate_sec")
        ), "net_clr_locksandthreads.contention_rate_sec missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="net_clr_locksandthreads.current_queue_length")
        ), "net_clr_locksandthreads.current_queue_length missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="net_clr_memory.num_bytes_in_all_heaps")
        ), "net_clr_memory.num_bytes_in_all_heaps missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="net_clr_memory.pct_time_in_gc")
        ), "net_clr_memory.pct_time_in_gc missing"
        assert wait_for(
            p(has_datapoint, backend, metric_name="net_clr_memory.num_gc_handles")
        ), "net_clr_memory.num_gc_handles"
        assert wait_for(
            p(has_datapoint, backend, metric_name="net_clr_memory.num_total_committed_bytes")
        ), "net_clr_memory.num_total_committed_bytes"
        assert wait_for(
            p(has_datapoint, backend, metric_name="net_clr_memory.num_total_reserved_bytes")
        ), "net_clr_memory.num_total_reserved_bytes"
        assert wait_for(
            p(has_datapoint, backend, metric_name="net_clr_memory.num_of_pinned_objects")
        ), "net_clr_memory.num_of_pinned_objects missing"

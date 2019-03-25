import os
import pathlib
import re
import string
import subprocess
from collections import namedtuple

Node = namedtuple("Node", ["flat", "flat_percent", "sum_percent", "cum", "cum_percent", "func", "src_line"])


class Profile(
    namedtuple(
        "Profile", ["nodes", "sampled", "percent_sampled", "total", "nodes_dropped", "cum_dropped", "original_output"]
    )
):
    pass


class PProfClient:
    """
    Expects that `go tool pprof` is available on the system.
    """

    def __init__(self, host: str, port: int):
        self.host = host
        self.port = port
        self.goroutine_idx = 0
        self.heap_idx = 0
        self.test_name = re.search(r"::(.*?) \(.*\)$", os.environ.get("PYTEST_CURRENT_TEST", "unknown")).group(1)

        pathlib.Path("/tmp/pprof").mkdir(parents=True, exist_ok=True)

    @property
    def _base_url(self):
        return f"http://{self.host}:{self.port}"

    def run_pprof(self, profile_type, sample_index=None, unit=""):
        sample_index_flag = ""
        if sample_index:
            sample_index_flag = "-sample_index=" + sample_index

        command = (
            f'go tool pprof -text -compact_labels -lines {sample_index_flag} -unit "{unit}" '
            + f"{self._base_url}/debug/pprof/{profile_type}"
        )

        profile_text = subprocess.check_output(command, shell=True)
        return self._parse_profile(profile_text)

    @staticmethod
    def _parse_profile(profile_output):
        lines = profile_output.decode("utf-8").splitlines()
        if not lines:
            return None

        sampled, percent_sampled, total = re.match(
            r"Showing nodes accounting for ([.\w]+), ([.\d]+)% of ([.\w]+) total", lines.pop(0)
        ).groups()

        nodes_dropped = "0"
        cum_dropped = "0"
        if lines[0].startswith("Dropped"):
            nodes_dropped, cum_dropped = re.match(r"Dropped (\d+) nodes \(cum <= ([.\w]+)\)", lines.pop(0)).groups()

        assert lines.pop(0).split() == ["flat", "flat%", "sum%", "cum", "cum%"], "unexpected pprof header line"

        nodes = [
            Node(*[float(c.strip(string.ascii_letters + "%")) for c in li.split()[:5]], *li.split()[5:]) for li in lines
        ]

        return Profile(
            sampled=float(sampled.strip(string.ascii_letters)),
            percent_sampled=float(percent_sampled),
            total=float(total.strip(string.ascii_letters)),
            nodes_dropped=nodes_dropped,
            cum_dropped=float(cum_dropped.strip(string.ascii_letters)),
            nodes=nodes,
            original_output=profile_output.decode("utf-8"),
        )

    def get_goroutine_profile(self):
        return self.run_pprof("goroutine")

    def get_heap_profile(self):
        return self.run_pprof("heap", "inuse_space")

    def assert_goroutine_count_under(self, count):
        from .util import assert_wait_for

        def check():
            check.last_profile = self.get_goroutine_profile()
            return check.last_profile.total < count

        assert_wait_for(
            check, interval_seconds=2, timeout_seconds=60, on_fail=lambda: print(check.last_profile.original_output)
        )

    def assert_heap_alloc_under(self, bytes_total):
        from .util import assert_wait_for

        def check():
            check.last_profile = self.get_heap_profile()
            return check.last_profile.total < (bytes_total * 1.5)

        assert_wait_for(
            check, interval_seconds=2, timeout_seconds=60, on_fail=lambda: print(check.last_profile.original_output)
        )

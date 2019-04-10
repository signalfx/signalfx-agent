import os
import tempfile
import time
from functools import partial as p
from textwrap import dedent

import psutil
import pytest
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import run_agent, wait_for

pytestmark = [pytest.mark.monitor_with_endpoints]


def script_path(script_name):
    return os.path.join(os.path.dirname(__file__), "scripts", script_name)


@pytest.mark.parametrize("script", [script_path("monitor1.py"), script_path("monitor2.py")])
def test_python_monitor_basics(script):
    config = dedent(
        f"""
            monitors:
              - type: python-monitor
                scriptFilePath: {script}
                intervalSeconds: 1
                a: test
            """
    )

    with run_agent(config) as [backend, _, _]:
        assert wait_for(
            p(has_datapoint, backend, metric_name="my.gauge", dimensions={"a": "test"}, count=5)
        ), "Didn't get datapoints"


def test_python_monitor_restarts_when_killed():
    config = dedent(
        f"""
            monitors:
              - type: python-monitor
                scriptFilePath: {script_path("monitor1.py")}
                intervalSeconds: 1
                a: test
            collectd:
              disableCollectd: true
            """
    )

    with run_agent(config, with_pid=True) as [backend, _, _, pid]:
        assert wait_for(
            p(has_datapoint, backend, metric_name="my.gauge", dimensions={"a": "test"}, count=2)
        ), "Didn't get datapoints"

        proc = psutil.Process(pid)
        assert len(proc.children()) == 1, "not exactly one subprocess"

        for _ in range(0, 5):
            proc.children()[0].terminate()

            time.sleep(1)
            backend.reset_datapoints()

            assert wait_for(
                p(has_datapoint, backend, metric_name="my.gauge", dimensions={"a": "test"}, count=2)
            ), "Didn't get datapoints"

            assert len(proc.children()) == 1, "not exactly one subprocess"


def test_python_monitor_respects_python_path():
    with tempfile.TemporaryDirectory() as tmpdir:
        with open(os.path.join(tmpdir, "randommodule.py"), "w") as fd:
            fd.write("print('hello')")

        config = dedent(
            f"""
                monitors:
                  - type: python-monitor
                    scriptFilePath: {script_path("monitor3.py")}
                    pythonPath:
                     - {tmpdir}
                    intervalSeconds: 1
                    a: test
                """
        )

        with run_agent(config) as [backend, _, _]:
            assert wait_for(
                p(has_datapoint, backend, metric_name="my.gauge", dimensions={"a": "test"}, count=5)
            ), "Didn't get datapoints"

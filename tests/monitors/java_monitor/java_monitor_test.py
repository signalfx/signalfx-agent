import os
import time
from functools import partial as p
from textwrap import dedent

import psutil
import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import wait_for

pytestmark = [pytest.mark.monitor_with_endpoints]


def mon_path(file_path):
    return os.path.join(os.path.dirname(__file__), "test_app", file_path)


def test_java_monitor_basics():
    config = dedent(
        f"""
            monitors:
              - type: java-monitor
                # This is a "fat jar" that has all dependencies bundled.
                jarFilePath: {mon_path("target/agent-test-app-1.0-SNAPSHOT.jar")}
                # Test that specifying the main class manually works.  This
                # test jar defines a main method in the manifest so this is
                # technically not necessary.
                mainClass: com.signalfx.agent.testmonitor.TestMonitor
                intervalSeconds: 1
                a: test
            """
    )

    with Agent.run(config) as agent:
        assert wait_for(
            p(has_datapoint, agent.fake_services, metric_name="my.gauge", dimensions={"a": "test"}, count=5)
        ), "Didn't get datapoints"


def test_java_monitor_restarts_when_killed():
    config = dedent(
        f"""
            monitors:
              - type: java-monitor
                jarFilePath: {mon_path("target/agent-test-app-1.0-SNAPSHOT.jar")}
                intervalSeconds: 1
                a: test
            collectd:
              disableCollectd: true
            """
    )

    with Agent.run(config) as agent:
        assert wait_for(
            p(has_datapoint, agent.fake_services, metric_name="my.gauge", dimensions={"a": "test"}, count=2)
        ), "Didn't get datapoints"

        proc = psutil.Process(agent.pid)
        assert len(proc.children()) == 1, "not exactly one subprocess"

        for _ in range(0, 5):
            proc.children()[0].terminate()

            time.sleep(1)
            agent.fake_services.reset_datapoints()

            assert wait_for(
                p(has_datapoint, agent.fake_services, metric_name="my.gauge", dimensions={"a": "test"}, count=2)
            ), "Didn't get datapoints"

            assert len(proc.children()) == 1, "not exactly one subprocess"

from functools import partial as p
from pathlib import Path
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, copy_file_content_into_container, run_container, wait_for
from tests.helpers.verify import verify

PIPELINE_CONF = Path(__file__).parent.joinpath("pipeline.conf").resolve()

SAMPLE_EVENTS = """
Logged in
Took 1 seconds
Logged in
Took 2 seconds
Logged in
Took 3 seconds
Logged in
Took 4 seconds
Logged in
Took 5 seconds
Logged in
Took 6 seconds
Logged in
Took 7 seconds
"""

METADATA = Metadata.from_package("logstash/logstash")


@pytest.mark.parametrize("version", ["7.3.0", "6.0.0"])
def test_logstash_tcp_client(version):
    with run_container(
        f"docker.elastic.co/logstash/logstash:{version}",
        environment={"XPACK_MONITORING_ENABLED": "false", "CONFIG_RELOAD_AUTOMATIC": "true"},
    ) as logstash_cont:
        copy_file_content_into_container(SAMPLE_EVENTS, logstash_cont, "tmp/events.log")
        copy_file_content_into_container(
            PIPELINE_CONF.read_text(), logstash_cont, "/usr/share/logstash/pipeline/test.conf"
        )
        host = container_ip(logstash_cont)

        config = dedent(
            f"""
            monitors:
              - type: logstash
                host: {host}
                port: 9600
            """
        )

        with Agent.run(config) as agent:
            assert wait_for(p(tcp_socket_open, host, 9600), timeout_seconds=120), "logstash didn't start"
            assert wait_for(
                p(has_datapoint, agent.fake_services, "node.stats.pipelines.events.in", value=15, dimensions={})
            )
            assert wait_for(
                p(has_datapoint, agent.fake_services, "node.stats.pipelines.events.out", value=15, dimensions={})
            )
            verify(agent, METADATA.default_metrics, 10)

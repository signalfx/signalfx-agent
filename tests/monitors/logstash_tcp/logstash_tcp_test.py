import re
from functools import partial as p
from pathlib import Path
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.util import (
    container_ip,
    copy_file_content_into_container,
    get_host_ip,
    run_container,
    wait_for,
    wait_for_value,
)

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


@pytest.mark.parametrize("version", ["7.3.0", "5.6.16"])
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
              - type: logstash-tcp
                mode: client
                host: {host}
                port: 8900
            """
        )

        with Agent.run(config) as agent:
            assert wait_for(p(tcp_socket_open, host, 8900), timeout_seconds=180), "logstash didn't start"
            assert wait_for(p(has_datapoint, agent.fake_services, "logins.count", value=7, dimensions={}))
            assert wait_for(p(has_datapoint, agent.fake_services, "process_time.count", value=7, dimensions={}))
            assert wait_for(p(has_datapoint, agent.fake_services, "process_time.mean", value=4, dimensions={}))


LISTEN_LOG_RE = re.compile(r"Listening for Logstash events on .*:(\d+)")


@pytest.mark.parametrize("version", ["7.3.0", "5.6.16"])
def test_logstash_tcp_server(version):
    with run_container(
        f"docker.elastic.co/logstash/logstash:{version}",
        environment={"XPACK_MONITORING_ENABLED": "false", "CONFIG_RELOAD_AUTOMATIC": "true"},
    ) as logstash_cont:
        agent_host = get_host_ip()

        copy_file_content_into_container(SAMPLE_EVENTS, logstash_cont, "tmp/events.log")

        config = dedent(
            f"""
            monitors:
              - type: logstash-tcp
                mode: server
                host: 0.0.0.0
                port: 0
            """
        )

        with Agent.run(config) as agent:
            log_match = wait_for_value(lambda: LISTEN_LOG_RE.search(agent.output))
            assert log_match is not None
            listen_port = int(log_match.groups()[0])

            copy_file_content_into_container(
                # The pipeline conf is written for server mode so patch it to
                # act as a client.
                PIPELINE_CONF.read_text()
                .replace('mode => "server"', 'mode => "client"')
                .replace('host => "0.0.0.0"', f'host => "{agent_host}"')
                .replace("port => 8900", f"port => {listen_port}"),
                logstash_cont,
                "/usr/share/logstash/pipeline/test.conf",
            )

            assert wait_for(
                p(has_datapoint, agent.fake_services, "logins.count", value=7, dimensions={}), timeout_seconds=180
            )
            assert wait_for(p(has_datapoint, agent.fake_services, "process_time.count", value=7, dimensions={}))
            assert wait_for(p(has_datapoint, agent.fake_services, "process_time.mean", value=4, dimensions={}))

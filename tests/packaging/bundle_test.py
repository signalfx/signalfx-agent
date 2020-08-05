from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import (
    container_ip,
    copy_file_content_into_container,
    copy_file_into_container,
    print_lines,
    run_service,
    wait_for,
)

from .common import is_agent_running_as_non_root, run_init_system_image

pytestmark = pytest.mark.bundle


def write_agent_config(cont, activemq_container):
    agent = Agent(
        host="127.0.0.1",
        fake_services=None,
        run_dir="/tmp/signalfx",
        config=dedent(
            f"""
        signalFxRealm: us0
        observers:
          - type: host

        monitors:
          - type: host-metadata
          - type: cpu
          - type: filesystems
          - type: disk-io
          - type: net-io
          - type: load
          - type: memory
          - type: vmem
          # This is a GenericJMX Java plugin, so we test the bundled Java runtime
          - type: collectd/activemq
            host: {container_ip(activemq_container)}
            port: 1099
            username: testuser
            password: testing123
    """
        ),
    )
    copy_file_content_into_container(agent.get_final_config_yaml(), cont, "/etc/signalfx/agent.yaml")


@pytest.mark.parametrize("base_image", ["alpine", "ubuntu1804"])
def test_bundle(request, base_image):
    # Get bundle path from command line flag to pytest
    bundle_path = request.config.getoption("--test-bundle-path")
    if not bundle_path:
        raise ValueError("You must specify the --test-bundle-path flag to run bundle tests")

    with run_service("activemq") as activemq_container:
        with run_init_system_image(base_image, command="/usr/bin/tail -f /dev/null") as [cont, backend]:
            copy_file_into_container(bundle_path, cont, "/opt/bundle.tar.gz")

            code, output = cont.exec_run(f"tar -xf /opt/bundle.tar.gz -C /opt")
            assert code == 0, f"Could not untar bundle: {output}"

            code, output = cont.exec_run(f"/opt/signalfx-agent/bin/patch-interpreter /opt/signalfx-agent")
            assert code == 0, f"Could not patch interpreter: {output}"

            write_agent_config(cont, activemq_container)

            _, output = cont.exec_run(
                ["/bin/sh", "-c", "exec /opt/signalfx-agent/bin/signalfx-agent > /var/log/signalfx-agent.log"],
                detach=True,
                stream=True,
            )

            try:
                assert wait_for(
                    p(has_datapoint, backend, metric_name="cpu.utilization"), timeout_seconds=10
                ), "Python metadata datapoint didn't come through"
                assert wait_for(
                    p(has_datapoint, backend, metric_name="gauge.amq.queue.QueueSize")
                ), "Didn't get activemq queue size datapoint"
                code, output = cont.exec_run("/opt/signalfx-agent/bin/agent-status")
                assert code == 0, f"failed to execute agent-status:\n{output.decode('utf-8')}"
            finally:
                print("Agent log:")
                _, output = cont.exec_run("cat /var/log/signalfx-agent.log")
                print_lines(output)


@pytest.mark.parametrize("base_image", ["ubuntu1804", "centos8"])
def test_bundle_non_root_user(request, base_image):
    # Get bundle path from command line flag to pytest
    bundle_path = request.config.getoption("--test-bundle-path")
    if not bundle_path:
        raise ValueError("You must specify the --test-bundle-path flag to run bundle tests")

    with run_service("activemq") as activemq_container:
        with run_init_system_image(base_image, command="/usr/bin/tail -f /dev/null") as [cont, backend]:
            code, output = cont.exec_run(
                "useradd --system --user-group --no-create-home --shell /sbin/nologin test-user"
            )
            assert code == 0, f"failed to create test-user:\n{output.decode('utf-8')}"

            copy_file_into_container(bundle_path, cont, "/opt/bundle.tar.gz")

            code, output = cont.exec_run(f"tar -xf /opt/bundle.tar.gz -C /opt")
            assert code == 0, f"Could not untar bundle: {output}"

            cont.exec_run("chown -R test-user:test-user /opt/signalfx-agent")

            code, output = cont.exec_run(
                f"/opt/signalfx-agent/bin/patch-interpreter /opt/signalfx-agent", user="test-user"
            )
            assert code == 0, f"Could not patch interpreter: {output}"

            write_agent_config(cont, activemq_container)

            cont.exec_run("chown test-user:test-user /etc/signalfx/agent.yaml")

            _, output = cont.exec_run(
                [
                    "/bin/sh",
                    "-c",
                    "exec /opt/signalfx-agent/bin/signalfx-agent > /opt/signalfx-agent/signalfx-agent.log",
                ],
                detach=True,
                stream=True,
                user="test-user",
            )

            try:
                assert is_agent_running_as_non_root(cont, user="test-user"), f"agent is not running as test-user"
                assert wait_for(
                    p(has_datapoint, backend, metric_name="cpu.utilization"), timeout_seconds=10
                ), "Python metadata datapoint didn't come through"
                assert wait_for(
                    p(has_datapoint, backend, metric_name="gauge.amq.queue.QueueSize")
                ), "Didn't get activemq queue size datapoint"
                code, output = cont.exec_run("/opt/signalfx-agent/bin/agent-status", user="test-user")
                assert code == 0, f"failed to execute agent-status:\n{output.decode('utf-8')}"
            finally:
                print("Agent log:")
                _, output = cont.exec_run("cat /opt/signalfx-agent/signalfx-agent.log")
                print_lines(output)

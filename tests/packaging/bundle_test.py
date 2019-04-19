from functools import partial as p
from textwrap import dedent

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import container_ip, print_lines, run_service, wait_for

from .common import copy_file_content_into_container, copy_file_into_container, run_init_system_image

pytestmark = pytest.mark.bundle


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
                  - type: collectd/cpu
                  - type: collectd/cpufreq
                  - type: collectd/df
                  - type: collectd/disk
                  - type: collectd/interface
                  - type: collectd/load
                  - type: collectd/memory
                  # This is a Python plugin, so we test the bundled Python runtime
                  - type: collectd/signalfx-metadata
                  - type: collectd/vmem
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

            _, output = cont.exec_run(
                ["/bin/sh", "-c", "exec /opt/signalfx-agent/bin/signalfx-agent > /var/log/signalfx-agent.log"],
                detach=True,
                stream=True,
            )

            try:
                assert wait_for(
                    p(has_datapoint, backend, dimensions={"plugin": "signalfx-metadata"}), timeout_seconds=10
                ), "Python metadata datapoint didn't come through"
                assert wait_for(
                    p(has_datapoint, backend, metric_name="gauge.amq.queue.QueueSize")
                ), "Didn't get activemq queue size datapoint"
            finally:
                print("Agent log:")
                _, output = cont.exec_run("cat /var/log/signalfx-agent.log")
                print_lines(output)

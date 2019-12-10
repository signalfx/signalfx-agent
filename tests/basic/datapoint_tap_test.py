from tests.helpers.agent import Agent
from tests.helpers.util import run_service, run_subprocess, wait_for
from tests.paths import AGENT_BIN

CONFIG = """
observers:
  - type: docker
monitors:
  - type: collectd/nginx
    discoveryRule: container_name =~ "nginx-dp-tap" && port == 80
"""


def test_basic_service_discovery():
    with Agent.run(CONFIG) as agent:
        with run_service("nginx", name="nginx-dp-tap"):
            with run_subprocess(
                command=[
                    str(AGENT_BIN),
                    "-config",
                    agent.config_path,
                    "tap-dps",
                    "-metric",
                    "nginx_connections*",
                    "-dims",
                    "{plugin: nginx}",
                ],
                close_fds=False,
            ) as [get_output, _]:

                def shows_datapoints():
                    return "nginx_connections.active" in get_output()

                wait_for(shows_datapoints, timeout_seconds=10)

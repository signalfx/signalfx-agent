import time
from functools import partial as p

from tests.config_sources.vault_test import run_vault
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, has_no_datapoint
from tests.helpers.util import wait_for


def test_reloads_new_config():
    with Agent.run(
        """
      monitors:
        - type: cpu
      """
    ) as agent:
        assert wait_for(
            p(has_datapoint, agent.fake_services, metric_name="cpu.utilization")
        ), "Didn't get kafka datapoints with properly mapped config"

        agent.update_config(
            """
        intervalSeconds: 1
        monitors:
         - type: memory
         - type: internal-metrics
        """
        )
        time.sleep(5)
        agent.fake_services.datapoints.clear()
        time.sleep(3)

        assert has_datapoint(agent.fake_services, metric_name="memory.utilization")
        assert has_datapoint(agent.fake_services, metric_name="sfxagent.active_monitors", value=2)
        assert has_no_datapoint(agent.fake_services, metric_name="cpu.utilization")


def test_maintains_old_config_if_new_is_bad():
    with Agent.run(
        """
      monitors:
        - type: cpu
        - type: internal-metrics
      """
    ) as agent:
        assert wait_for(
            p(has_datapoint, agent.fake_services, metric_name="cpu.utilization")
        ), "Didn't get kafka datapoints with properly mapped config"

        agent.update_config(
            """
        intervalSeconds: 1
        monitors: {}
        """
        )
        time.sleep(5)
        agent.fake_services.datapoints.clear()
        time.sleep(3)

        assert has_datapoint(agent.fake_services, metric_name="cpu.utilization")
        assert has_datapoint(agent.fake_services, metric_name="sfxagent.active_monitors", value=2)


def test_reloads_new_remote_config():
    with run_vault() as [vault_client, _]:
        vault_client.sys.enable_secrets_engine(backend_type="kv", options={"version": "1"})

        vault_client.write("kv/extradims", data={"env": "prod"})

        with Agent.run(
            f"""
            globalDimensions:
             env: {{"#from": "vault:kv/extradims[data.env]"}}
            configSources:
              vault:
                vaultToken: {vault_client.token}
                vaultAddr: {vault_client.url}
                kvV2PollInterval: 1s

            monitors:
             - type: cpu
          """
        ) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"env": "prod"}))

            vault_client.write("kv/extradims", data={"env": "dev"})

            time.sleep(5)
            agent.fake_services.datapoints.clear()
            time.sleep(3)

            assert has_datapoint(agent.fake_services, dimensions={"env": "dev"})
            assert has_no_datapoint(agent.fake_services, dimensions={"env": "prod"})

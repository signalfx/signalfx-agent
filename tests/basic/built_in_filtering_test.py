from tests.helpers.agent import Agent
from tests.helpers.metadata import Metadata
from tests.helpers.verify import verify
from tests.monitors.expvar.expvar_test import run_expvar
from tests.monitors.redis.redis_test import run_redis
from tests.paths import REPO_ROOT_DIR


def test_extra_metrics_passthrough():
    """
    The specified extraMetrics should be allowed through even though they are
    not by default.
    """
    metadata = Metadata.from_package("expvar")

    with run_expvar() as expvar_container_ip:
        with Agent.run(
            f"""
               monitors:
                 - type: expvar
                   host: {expvar_container_ip}
                   port: 8080
                   intervalSeconds: 1
                   extraMetrics:
                    - memstats.by_size.mallocs
               """
        ) as agent:
            assert "memstats.by_size.mallocs" in metadata.nondefault_metrics
            verify(agent, metadata.default_metrics | {"memstats.by_size.mallocs"})


def test_built_in_filtering_disabled_no_whitelist_for_monitor():
    """
    Test a monitor that doesn't have any entries in whitelist.json
    """
    metadata = Metadata.from_package("expvar")

    with run_expvar() as expvar_container_ip:
        with Agent.run(
            f"""
               enableBuiltInFiltering: false
               monitors:
                 - type: expvar
                   host: {expvar_container_ip}
                   port: 8080
                   intervalSeconds: 1
                   enhancedMetrics: true
                   # This should be ignored
                   extraMetrics:
                    - memstats.by_size.mallocs
               metricsToExclude:
                - {{"#from": "{REPO_ROOT_DIR}/whitelist.json", flatten: true}}
               """
        ) as agent:
            verify(agent, metadata.all_metrics)


def test_built_in_filtering_disabled_whitelisted_monitor():
    """
    Test a monitor that is in whitelist.json.
    """
    metadata = Metadata.from_package("collectd/redis")

    with run_redis() as [ip_addr, redis_client]:
        redis_client.lpush("queue-1", *["a", "b", "c"])
        redis_client.lpush("queue-2", *["x", "y"])

        with Agent.run(
            f"""
               enableBuiltInFiltering: false
               monitors:
                 - type: collectd/redis
                   host: {ip_addr}
                   port: 6379
                   intervalSeconds: 1
                   sendListLengths:
                    - databaseIndex: 0
                      keyPattern: queue-*
               metricsToExclude:
                - {{"#from": "{REPO_ROOT_DIR}/whitelist.json", flatten: true}}
               """
        ) as agent:
            key_llen_metric = "gauge.key_llen"
            assert key_llen_metric not in metadata.default_metrics
            verify(agent, metadata.default_metrics - {"gauge.slave_repl_offset"})

            # Add a non-default metric to the whitelist via metricsToInclude
            # and make sure it comes through
            agent.config["metricsToInclude"] = [{"monitorType": "collectd/redis", "metricName": key_llen_metric}]
            agent.write_config()
            verify(agent, metadata.default_metrics - {"gauge.slave_repl_offset"} | {key_llen_metric})

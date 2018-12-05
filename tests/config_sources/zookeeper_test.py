import string
from functools import partial as p

from helpers.assertions import container_cmd_exit_0, has_datapoint_with_dim
from helpers.util import container_ip, run_agent, run_container, wait_for

CONFIG = string.Template(
    """
globalDimensions:
  env: {"#from": "zookeeper:/env"}
configSources:
  zookeeper:
    endpoints:
    - $zk_endpoint
monitors:
 - { "#from": "zookeeper:/monitors/*", flatten: true }
"""
)

BAD_GLOB_CONFIG = string.Template(
    """
globalDimensions:
  env: {"#from": "zookeeper:/env"}
configSources:
  zookeeper:
    endpoints:
    - $zk_endpoint
monitors:
 # Non-terminating globs are not allowed!
 - { "#from": "zookeeper:/*/monitors", flatten: true }
"""
)


def create_znode(container, path, value):
    container.exec_run("zkCli.sh create %s '%s'" % (path, value))


def test_basic_zk_config():
    with run_container("zookeeper:3.4") as zk_cont:
        assert wait_for(p(container_cmd_exit_0, zk_cont, "nc -z localhost 2181"), 5)
        create_znode(zk_cont, "/env", "prod")
        create_znode(zk_cont, "/monitors", "")
        create_znode(zk_cont, "/monitors/cpu", "- type: collectd/cpu")
        create_znode(zk_cont, "/monitors/signalfx-metadata", "- type: collectd/signalfx-metadata")

        final_conf = CONFIG.substitute(zk_endpoint="%s:2181" % container_ip(zk_cont))
        with run_agent(final_conf) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata"))
            assert wait_for(p(has_datapoint_with_dim, backend, "env", "prod"))


def test_bad_globbing():
    with run_container("zookeeper:3.4") as zk_cont:
        assert wait_for(p(container_cmd_exit_0, zk_cont, "nc -z localhost 2181"), 5)
        create_znode(zk_cont, "/env", "prod")

        final_conf = BAD_GLOB_CONFIG.substitute(zk_endpoint="%s:2181" % container_ip(zk_cont))
        with run_agent(final_conf) as [_, get_output, _]:
            assert wait_for(lambda: "Zookeeper only supports globs" in get_output())

import string
from functools import partial as p

from tests.helpers.assertions import has_datapoint_with_dim, tcp_socket_open
from tests.helpers.util import container_ip, run_agent, run_container, wait_for

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
        zkhost = container_ip(zk_cont)
        assert wait_for(p(tcp_socket_open, zkhost, 2181), 30)
        create_znode(zk_cont, "/env", "prod")
        create_znode(zk_cont, "/monitors", "")
        create_znode(zk_cont, "/monitors/cpu", "- type: collectd/cpu")
        create_znode(zk_cont, "/monitors/signalfx-metadata", "- type: collectd/signalfx-metadata")

        final_conf = CONFIG.substitute(zk_endpoint="%s:2181" % zkhost)
        with run_agent(final_conf) as [backend, _, _]:
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata"))
            assert wait_for(p(has_datapoint_with_dim, backend, "env", "prod"))


def test_bad_globbing():
    with run_container("zookeeper:3.4") as zk_cont:
        zkhost = container_ip(zk_cont)
        assert wait_for(p(tcp_socket_open, zkhost, 2181), 30)
        create_znode(zk_cont, "/env", "prod")

        final_conf = BAD_GLOB_CONFIG.substitute(zk_endpoint="%s:2181" % zkhost)
        with run_agent(final_conf) as [_, get_output, _]:
            assert wait_for(lambda: "zookeeper only supports globs" in get_output())

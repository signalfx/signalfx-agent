from functools import partial as p
import os
import string

from tests.helpers import fake_backend
from tests.helpers.util import (
    container_ip,
    wait_for,
    run_agent,
    run_container
)
from tests.helpers.assertions import *

config = string.Template("""
globalDimensions:
  env: {"#from": "zookeeper:/env"}
configSources:
  zookeeper:
    endpoints:
    - $zk_endpoint
monitors:
 - { "#from": "zookeeper:/monitors/*", flatten: true }
""")

bad_glob_config = string.Template("""
globalDimensions:
  env: {"#from": "zookeeper:/env"}
configSources:
  zookeeper:
    endpoints:
    - $zk_endpoint
monitors:
 # Non-terminating globs are not allowed!
 - { "#from": "zookeeper:/*/monitors", flatten: true }
""")


def create_znode(container, path, value):
    _, output = container.exec_run("zkCli.sh create %s '%s'" % (path, value))


def test_basic_zk_config():
    with run_container("zookeeper:3.4") as zk:
        wait_for(p(container_cmd_exit_0, zk, "nc -z localhost 2181"), 5)
        create_znode(zk, "/env", "prod")
        create_znode(zk, "/monitors", "")
        create_znode(zk, "/monitors/cpu", "- type: collectd/cpu")
        create_znode(zk, "/monitors/signalfx-metadata", "- type: collectd/signalfx-metadata")

        final_conf = config.substitute(zk_endpoint="%s:2181" % container_ip(zk))
        with run_agent(final_conf) as [backend, get_output]:
            wait_for(p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata"))
            wait_for(p(has_datapoint_with_dim, backend, "env", "prod"))


def test_bad_globbing():
    with run_container("zookeeper:3.4") as zk:
        wait_for(p(container_cmd_exit_0, zk, "nc -z localhost 2181"), 5)
        create_znode(zk, "/env", "prod")

        final_conf = bad_glob_config.substitute(zk_endpoint="%s:2181" % container_ip(zk))
        with run_agent(final_conf) as [backend, get_output]:
            wait_for(lambda: "Zookeeper only supports globs" in get_output())

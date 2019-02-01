import string
from functools import partial as p

from tests.helpers.assertions import container_cmd_exit_0, has_datapoint_with_dim, has_event_with_dim
from tests.helpers.util import container_ip, run_agent, run_container, wait_for


def create_path(container, path, value):
    _, output = container.exec_run("/etcdctl set -- %s '%s'" % (path, value))
    print("etcd: %s" % output)


ETCD2_IMAGE = "quay.io/coreos/etcd:v2.3.8"
# etcd takes a bit of coaxing to listen on anything other than localhost
ETCD_COMMAND = (
    "-listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 "
    "-advertise-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001"
)

CONFIG = string.Template(
    """
globalDimensions:
  env: {"#from": "etcd2:/env"}
  unit: {"#from": "etcd2:/unit", optional: true}
configSources:
  etcd2:
    endpoints:
    - http://$endpoint
monitors:
 - { "#from": "etcd2:/monitors/*", flatten: true }
"""
)


def test_basic_etcd2_config():
    with run_container(ETCD2_IMAGE, command=ETCD_COMMAND) as etcd:
        assert wait_for(p(container_cmd_exit_0, etcd, "/etcdctl ls"), 5), "etcd didn't start"
        create_path(etcd, "/env", "prod")
        create_path(etcd, "/monitors/cpu", "- type: collectd/cpu")
        create_path(etcd, "/monitors/signalfx-metadata", "- type: collectd/signalfx-metadata")

        final_conf = CONFIG.substitute(endpoint="%s:2379" % container_ip(etcd))
        with run_agent(final_conf) as [backend, _, _]:
            assert wait_for(
                p(has_datapoint_with_dim, backend, "plugin", "signalfx-metadata")
            ), "Datapoints didn't come through"
            assert wait_for(p(has_datapoint_with_dim, backend, "env", "prod")), "dimension wasn't set"


INTERNAL_GLOB_CONFIG = string.Template(
    """
configSources:
  etcd2:
    endpoints:
    - http://$endpoint
monitors:
 - { "#from": "etcd2:/services/*/monitor", flatten: true }
"""
)


def test_interior_globbing():
    with run_container(ETCD2_IMAGE, command=ETCD_COMMAND) as etcd:
        assert wait_for(p(container_cmd_exit_0, etcd, "/etcdctl ls"), 5), "etcd didn't start"
        create_path(etcd, "/env", "prod")
        create_path(etcd, "/services/cpu/monitor", "- type: collectd/cpu")
        create_path(etcd, "/services/signalfx/monitor", "- type: collectd/signalfx-metadata")

        final_conf = INTERNAL_GLOB_CONFIG.substitute(endpoint="%s:2379" % container_ip(etcd))
        with run_agent(final_conf) as [backend, _, _]:
            assert wait_for(
                p(has_event_with_dim, backend, "plugin", "signalfx-metadata")
            ), "Datapoints didn't come through"

            create_path(etcd, "/services/uptime/monitor", "- type: collectd/uptime")
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "uptime")), "didn't get uptime datapoints"

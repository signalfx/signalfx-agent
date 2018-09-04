"""
Integration tests for the docker observer
"""
import time
from functools import partial as p

from tests.helpers.assertions import has_datapoint_with_dim, has_log_message
from tests.helpers.util import ensure_always, run_agent, run_service, wait_for

CONFIG = """
observers:
  - type: docker
    labelsToDimensions:
      mylabel: mydim

monitors:
  - type: collectd/nginx
    discoveryRule: container_name =~ "nginx-discovery" && port == 80
    intervalSeconds: 1
"""


def test_docker_observer():
    with run_agent(CONFIG) as [backend, get_output, _]:
        with run_service("nginx", name="nginx-discovery", labels={"mylabel": "abc"}):
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "nginx")), "Didn't get nginx datapoints"
            assert wait_for(p(has_datapoint_with_dim, backend, "mydim", "abc")), "Didn't get custom label dimension"
        # Let nginx be removed by docker observer and collectd restart
        time.sleep(5)
        backend.datapoints.clear()
        assert ensure_always(lambda: not has_datapoint_with_dim(backend, "plugin", "nginx"), 10)
        assert not has_log_message(get_output(), "error")

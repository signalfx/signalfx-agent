import time
from functools import partial as p

from helpers.assertions import has_datapoint_with_dim, has_log_message
from helpers.util import ensure_always, run_agent, run_service, wait_for

CONFIG = """
observers:
  - type: docker
monitors:
  - type: collectd/nginx
    discoveryRule: container_name =~ "nginx-discovery" && port == 80
"""


def test_endpoint_config_mapping():
    with run_agent(CONFIG) as [backend, get_output, _]:
        with run_service("nginx", name="nginx-discovery",
                labels={"com.signalfx.extraDimensions": "{a: 1}"):
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "nginx")), "Didn't get nginx datapoints"
        # Let nginx be removed by docker observer and collectd restart
        time.sleep(5)
        backend.datapoints.clear()
        assert ensure_always(lambda: not has_datapoint_with_dim(backend, "plugin", "nginx"), 10)
        assert not has_log_message(get_output(), "error")

"""
Integration tests for the docker observer
"""
import time
from functools import partial as p
from textwrap import dedent
import string

from tests.helpers.assertions import has_datapoint_with_dim
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

HOST_BINDING_CONFIG = string.Template(
    """
observers:
  - type: docker
    useHostBindings: true
    ignoreNonHostBindings: true
    labelsToDimensions:
      mylabel: mydim

monitors:
  - type: collectd/nginx
    discoveryRule: container_name =~ "nginx-non-host-binding" && port == 80
    intervalSeconds: 1
  - type: collectd/nginx
    discoveryRule: container_name =~ "nginx-with-host-binding" && private_port == 80 && public_port == $port
    intervalSeconds: 1
"""
)


def test_docker_observer():
    with run_agent(CONFIG) as [backend, _, _]:
        with run_service("nginx", name="nginx-discovery", labels={"mylabel": "abc"}):
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "nginx")), "Didn't get nginx datapoints"
            assert wait_for(p(has_datapoint_with_dim, backend, "mydim", "abc")), "Didn't get custom label dimension"
        # Let nginx be removed by docker observer and collectd restart
        time.sleep(5)
        backend.datapoints.clear()
        assert ensure_always(lambda: not has_datapoint_with_dim(backend, "container_name", "nginx-discovery"), 10)


def test_docker_observer_use_host_bindings():
    with run_service("nginx", name="nginx-non-host-binding", labels={"mylabel": "non-host-binding"}):
        with run_service(
            "nginx",
            name="nginx-with-host-binding",
            labels={"mylabel": "with-host-binding"},
            ports={"80/tcp": ("127.0.0.1", 0)},
        ) as container_bind:
            with run_agent(
                HOST_BINDING_CONFIG.substitute(
                    port=container_bind.attrs["NetworkSettings"]["Ports"]["80/tcp"][0]["HostPort"]
                )
            ) as [backend, _, _]:
                assert not wait_for(
                    p(has_datapoint_with_dim, backend, "mydim", "non-host-binding")
                ), "Didn't get custom label dimension"
                assert wait_for(
                    p(has_datapoint_with_dim, backend, "mydim", "with-host-binding")
                ), "Didn't get custom label dimension"


def test_docker_observer_labels():
    """
    Test that docker observer picks up a fully configured endpoint from
    container labels
    """
    with run_agent(
        dedent(
            """
        observers:
          - type: docker
    """
        )
    ) as [backend, _, _]:
        with run_service(
            "nginx",
            name="nginx-disco-full",
            labels={
                "agent.signalfx.com.monitorType.80": "collectd/nginx",
                "agent.signalfx.com.config.80.intervalSeconds": "1",
            },
        ):
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "nginx")), "Didn't get nginx datapoints"
        # Let nginx be removed by docker observer and collectd restart
        time.sleep(5)
        backend.datapoints.clear()
        assert ensure_always(lambda: not has_datapoint_with_dim(backend, "container_name", "nginx-disco-full"), 10)


def test_docker_observer_labels_partial():
    """
    Test that docker observer picks up a partially configured endpoint from
    container labels
    """
    with run_agent(
        dedent(
            """
        observers:
          - type: docker
        monitors:
          - type: collectd/nginx
            discoveryRule: container_name =~ "nginx-disco-partial" && port == 80
    """
        )
    ) as [backend, _, _]:
        with run_service(
            "nginx",
            name="nginx-disco-partial",
            labels={"agent.signalfx.com.config.80.extraDimensions": "{mydim: myvalue}"},
        ):
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "nginx")), "Didn't get nginx datapoints"
            assert wait_for(p(has_datapoint_with_dim, backend, "mydim", "myvalue")), "Didn't get extra dimension"
        # Let nginx be removed by docker observer and collectd restart
        time.sleep(5)
        backend.datapoints.clear()
        assert ensure_always(lambda: not has_datapoint_with_dim(backend, "container_name", "nginx-disco-partial"), 10)


def test_docker_observer_labels_multiple_monitors_per_port():
    """
    Test that we can configure multiple monitors per port using labels
    """
    with run_agent(
        dedent(
            """
        observers:
          - type: docker
    """
        )
    ) as [backend, _, _]:
        with run_service(
            "nginx",
            name="nginx-multi-monitors",
            labels={
                "agent.signalfx.com.monitorType.80": "collectd/nginx",
                "agent.signalfx.com.config.80.intervalSeconds": "1",
                "agent.signalfx.com.config.80.extraDimensions": "{app: nginx}",
                "agent.signalfx.com.monitorType.80-nginx2": "collectd/nginx",
                "agent.signalfx.com.config.80-nginx2.intervalSeconds": "1",
                "agent.signalfx.com.config.80-nginx2.extraDimensions": "{app: other}",
            },
        ):
            assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "nginx")), "Didn't get nginx datapoints"
            assert wait_for(p(has_datapoint_with_dim, backend, "app", "nginx")), "Didn't get extra dims"
            assert wait_for(p(has_datapoint_with_dim, backend, "app", "other")), "Didn't get extra dims"
        # Let nginx be removed by docker observer and collectd restart
        time.sleep(5)
        backend.datapoints.clear()
        assert ensure_always(lambda: not has_datapoint_with_dim(backend, "container_name", "nginx-multi-monitors"), 10)

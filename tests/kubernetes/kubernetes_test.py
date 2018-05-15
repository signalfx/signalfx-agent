from tests.helpers import fake_backend
from tests.kubernetes.data import *
from tests.kubernetes.utils import *
import os
import pytest
import time

pytestmark = [pytest.mark.k8s, pytest.mark.kubernetes]

AGENT_YAMLS_DIR = os.environ.get("AGENT_YAMLS_DIR", "/go/src/github.com/signalfx/signalfx-agent/deployments/k8s")
AGENT_CONFIGMAP_PATH = os.environ.get("AGENT_CONFIGMAP_PATH", os.path.join(AGENT_YAMLS_DIR, "configmap.yaml"))
AGENT_DAEMONSET_PATH = os.environ.get("AGENT_DAEMONSET_PATH", os.path.join(AGENT_YAMLS_DIR, "daemonset.yaml"))
AGENT_SERVICEACCOUNT_PATH = os.environ.get("AGENT_SERVICEACCOUNT_PATH", os.path.join(AGENT_YAMLS_DIR, "serviceaccount.yaml"))
AGENT_IMAGE_NAME = os.environ.get("AGENT_IMAGE_NAME", "localhost:5000/signalfx-agent")
AGENT_IMAGE_TAG = os.environ.get("AGENT_IMAGE_TAG", "k8s-test")
DOCS_DIR = os.environ.get("DOCS_DIR", "/go/src/github.com/signalfx/signalfx-agent/docs")


@pytest.mark.parametrize(
    "monitor", 
    MONITORS_WITHOUT_ENDPOINTS,
    ids=[m["type"] for m in MONITORS_WITHOUT_ENDPOINTS])
def test_monitor_without_observer(minikube, monitor, k8s_test_timeout):
    if monitor["type"] in ["collectd/cpufreq", "collectd/df"]:
        pytest.skip("monitor %s not supported" % monitor["type"])
    monitor_doc = os.path.join(DOCS_DIR, "monitors", monitor["type"].replace("/", "-") + ".md")
    expected_metrics = get_metrics_from_doc(monitor_doc)
    expected_dims = get_dims_from_doc(monitor_doc)
    if len(expected_metrics) == 0 and len(expected_dims) == 0:
        pytest.skip("expected metrics and dimensions lists are empty")
    print("\nCollected %d metric(s) and %d dimension(s) to test for %s." % (len(expected_metrics), len(expected_dims), monitor["type"]))
    monitors = [monitor]
    if monitor["type"] == "collectd/cpu":
        monitors.append({"type": "collectd/signalfx-metadata"})
    elif monitor["type"] == "collectd/signalfx-metadata":
        monitors.append({"type": "collectd/cpu"})
    with fake_backend.start(ip=get_host_ip()) as backend:
        with minikube.deploy_agent(
            AGENT_CONFIGMAP_PATH,
            AGENT_DAEMONSET_PATH,
            AGENT_SERVICEACCOUNT_PATH,
            observer=None,
            monitors=monitors,
            cluster_name="minikube",
            backend=backend,
            image_name=AGENT_IMAGE_NAME,
            image_tag=AGENT_IMAGE_TAG,
            namespace="default") as agent:
            if len(expected_metrics) > 0 and len(expected_dims) > 0:
                assert any_metric_has_any_dim(backend, expected_metrics, expected_dims, k8s_test_timeout), \
                    "timed out waiting for any metric in %s with any dimension in %s!\n\nAGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n" % \
                    (expected_metrics, expected_dims, agent.get_status(), agent.get_container_logs())
            elif len(expected_metrics) > 0:
                assert any_metric_found(backend, expected_metrics, k8s_test_timeout), \
                    "timed out waiting for any metric in %s!\n\nAGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n" % \
                    (expected_metrics, agent.get_status(), agent.get_container_logs())
            else:
                assert any_dim_found(backend, expected_dims, k8s_test_timeout), \
                    "timed out waiting for any dimension in %s!\n\nAGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n" % \
                    (expected_dims, agent.get_status(), agent.get_container_logs())


@pytest.mark.parametrize(
    "monitor,yamls",
    MONITORS_WITH_ENDPOINTS,
    ids=[m[0]["type"] for m in MONITORS_WITH_ENDPOINTS])
def test_monitor_with_observer(minikube, monitor, yamls, k8s_observer, k8s_test_timeout):
    monitor_doc = os.path.join(DOCS_DIR, "monitors", monitor["type"].replace("/", "-") + ".md")
    observer_doc = os.path.join(DOCS_DIR, "observers", k8s_observer + ".md")
    expected_metrics = get_metrics_from_doc(monitor_doc)
    expected_dims = get_dims_from_doc(monitor_doc) + get_dims_from_doc(observer_doc)
    if monitor["type"] == "collectd/genericjmx" and len(expected_metrics) == 0:
        with open(os.path.join(os.path.dirname(os.path.realpath(__file__)), "genericjmx-metrics.txt"), "r") as fd:
            expected_metrics = [m.strip() for m in fd.readlines()]
    elif monitor["type"] == "collectd/health-checker" and len(expected_metrics) == 0:
        expected_metrics = ["gauge.service.health.status", "gauge.service.health.value"]
    elif monitor["type"] == "prometheus-exporter" and len(expected_metrics) == 0:
        with open(os.path.join(os.path.dirname(os.path.realpath(__file__)), "prometheus-metrics.txt"), "r") as fd:
            expected_metrics = [m.strip() for m in fd.readlines()]
    if len(expected_metrics) == 0 and len(expected_dims) == 0:
        pytest.skip("expected metrics and dimensions lists are empty")
    if len(yamls) == 0:
        pytest.skip("yamls list is empty")
    print("\nCollected %d metric(s) and %d dimension(s) to test for %s." % (len(expected_metrics), len(expected_dims), monitor["type"]))
    with fake_backend.start(ip=get_host_ip()) as backend:
        with minikube.deploy_yamls(yamls=yamls):
            with minikube.deploy_agent(
                AGENT_CONFIGMAP_PATH,
                AGENT_DAEMONSET_PATH,
                AGENT_SERVICEACCOUNT_PATH,
                observer=k8s_observer,
                monitors=[monitor],
                cluster_name="minikube",
                backend=backend,
                image_name=AGENT_IMAGE_NAME,
                image_tag=AGENT_IMAGE_TAG,
                namespace="default") as agent:
                if len(expected_metrics) > 0 and len(expected_dims) > 0:
                    assert any_metric_has_any_dim(backend, expected_metrics, expected_dims, k8s_test_timeout), \
                        "timed out waiting for any metric in %s with any dimension in %s!\n\nAGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n" % \
                        (expected_metrics, expected_dims, agent.get_status(), agent.get_container_logs())
                elif len(expected_metrics) > 0:
                    assert any_metric_found(backend, expected_metrics, k8s_test_timeout), \
                        "timed out waiting for any metric in %s!\n\nAGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n" % \
                        (expected_metrics, agent.get_status(), agent.get_container_logs())
                else:
                    assert any_dim_found(backend, expected_dims, k8s_test_timeout), \
                        "timed out waiting for any dimension in %s!\n\nAGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n" % \
                        (expected_dims, agent.get_status(), agent.get_container_logs())


def test_plaintext_passwords(minikube):
    with fake_backend.start(ip=get_host_ip()) as backend:
        with minikube.deploy_agent(
            AGENT_CONFIGMAP_PATH,
            AGENT_DAEMONSET_PATH,
            AGENT_SERVICEACCOUNT_PATH,
            observer=None,
            monitors=MONITORS_WITHOUT_ENDPOINTS + [m[0] for m in MONITORS_WITH_ENDPOINTS],
            cluster_name="minikube",
            backend=backend,
            image_name=AGENT_IMAGE_NAME,
            image_tag=AGENT_IMAGE_TAG,
            namespace="default") as agent:
            status = agent.get_status()
            assert "testing123" not in status, "plaintext password(s) found in agent-status output!\n\n%s\n" % status


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
CUR_DIR = os.path.dirname(os.path.realpath(__file__))


def test_monitor_without_endpoints(k8s_monitor_without_endpoints, k8s_test_timeout, minikube):
    monitor = k8s_monitor_without_endpoints
    monitors = [monitor]
    if monitor["type"] == "collectd/cpu":
        monitors.append({"type": "collectd/signalfx-metadata"})
    elif monitor["type"] == "collectd/signalfx-metadata":
        monitors.append({"type": "collectd/cpu"})
    if monitor["type"] == "collectd/docker":
        expected_metrics = get_metrics_from_doc(os.path.join(DOCS_DIR, "monitors", "docker-container-stats.md"))
        expected_dims = get_dims_from_doc(os.path.join(DOCS_DIR, "monitors", "docker-container-stats.md"))
    elif monitor["type"] == "collectd/statsd":
        expected_metrics = {"gauge.statsd.test"}
        expected_dims = {"foo", "dim"}
    else:
        monitor_doc = os.path.join(DOCS_DIR, "monitors", monitor["type"].replace("/", "-") + ".md")
        expected_metrics = get_metrics_from_doc(monitor_doc)
        expected_dims = get_dims_from_doc(monitor_doc)
    observer_doc = os.path.join(DOCS_DIR, "observers", "k8s-api.md")
    expected_dims = expected_dims.union(get_dims_from_doc(observer_doc), {"kubernetes_cluster"})
    metrics_txt = os.path.join(CUR_DIR, monitor["type"].replace("/", "-") + '-metrics.txt')
    if len(expected_metrics) == 0 and os.path.isfile(metrics_txt):
        with open(metrics_txt, "r") as fd:
            expected_metrics = {m.strip() for m in fd.readlines() if len(m.strip()) > 0}
    with fake_backend.start(ip=get_host_ip()) as backend:
        with minikube.deploy_agent(
            AGENT_CONFIGMAP_PATH,
            AGENT_DAEMONSET_PATH,
            AGENT_SERVICEACCOUNT_PATH,
            observer="k8s-api",
            monitors=monitors,
            cluster_name="minikube",
            backend=backend,
            image_name=AGENT_IMAGE_NAME,
            image_tag=AGENT_IMAGE_TAG,
            namespace="default") as agent:
            if monitor["type"] == "collectd/statsd":
                agent.container.exec_run(["/bin/bash", "-c", 'while true; do echo "statsd.[foo=bar,dim=val]test:1|g" | nc -w 1 -u 127.0.0.1 8125; sleep 1; done'], detach=True)
            if monitor["type"] not in ["collectd/cpufreq", "collectd/custom", "kubernetes-events"]:
                print("\nCollected %d metric(s) and %d dimension(s) to test for %s." % (len(expected_metrics), len(expected_dims), monitor["type"]))
                if len(expected_metrics) > 0 and len(expected_dims) > 0:
                    assert any_metric_has_any_dim(backend, expected_metrics, expected_dims, k8s_test_timeout), \
                        "timed out waiting for any metric in %s with any dimension in %s!\n\nAGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n" % \
                        (expected_metrics, expected_dims, agent.get_status(), agent.get_container_logs())
                elif len(expected_metrics) > 0:
                    assert any_metric_found(backend, expected_metrics, k8s_test_timeout), \
                        "timed out waiting for any metric in %s!\n\nAGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n" % \
                        (expected_metrics, agent.get_status(), agent.get_container_logs())
                elif len(expected_dims) > 0:
                    assert any_dim_found(backend, expected_dims, k8s_test_timeout), \
                        "timed out waiting for any dimension in %s!\n\nAGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n" % \
                        (expected_dims, agent.get_status(), agent.get_container_logs())


def test_monitor_with_endpoints(k8s_monitor_with_endpoints, k8s_observer, k8s_test_timeout, minikube):
    monitor, yamls = k8s_monitor_with_endpoints
    monitor_doc = os.path.join(DOCS_DIR, "monitors", monitor["type"].replace("/", "-") + ".md")
    observer_doc = os.path.join(DOCS_DIR, "observers", k8s_observer + ".md")
    metrics_txt = os.path.join(CUR_DIR, monitor["type"].replace("/", "-") + '-metrics.txt')
    expected_metrics = get_metrics_from_doc(monitor_doc)
    expected_dims = get_dims_from_doc(monitor_doc).union(get_dims_from_doc(observer_doc), {"kubernetes_cluster"})
    if len(expected_metrics) == 0 and os.path.isfile(metrics_txt):
        with open(metrics_txt, "r") as fd:
            expected_metrics = {m.strip() for m in fd.readlines() if len(m.strip()) > 0}
    assert len(expected_metrics) > 0 and len(expected_dims) > 0, "expected metrics and dimensions lists are both empty!"
    assert len(yamls) > 0, "yamls list is empty!"
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
                print("\nCollected %d metric(s) and %d dimension(s) to test for %s." % (len(expected_metrics), len(expected_dims), monitor["type"]))
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
            observer="k8s-api",
            monitors=MONITORS_WITHOUT_ENDPOINTS + [m[0] for m in MONITORS_WITH_ENDPOINTS],
            cluster_name="minikube",
            backend=backend,
            image_name=AGENT_IMAGE_NAME,
            image_tag=AGENT_IMAGE_TAG,
            namespace="default") as agent:
            agent_status = agent.get_status()
            container_logs = agent.get_container_logs()
            assert "testing123" not in agent_status, "plaintext password(s) found in agent-status output!\n\n%s\n" % agent_status
            assert "testing123" not in container_logs, "plaintext password(s) found in agent container logs!\n\n%s\n" % container_logs


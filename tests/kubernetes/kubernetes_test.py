from functools import partial as p
from tests.helpers.util import *
from tests.helpers.assertions import *
from tests.kubernetes.utils import *

import os
import pytest

pytestmark = [pytest.mark.k8s, pytest.mark.kubernetes]

SERVICES_TO_DEPLOY = ["nginx"]

# list of tuples for the monitors and respective metrics to test
# the monitor should be a YAML-based dictionary which will be used for the signalfx-agent agent.yaml configuration
MONITORS_TO_TEST = [
    ({"type": "collectd/cpu"}, get_metrics_from_doc("collectd-cpu.md")),
    #({"type": "collectd/cpufreq"}, ["cpufreq.0"]),
    #({"type": "collectd/df", "hostFSPath": "/hostfs", "reportInodes": True}, get_metrics_from_doc("collectd-df.md")),
    ({"type": "collectd/interface"}, get_metrics_from_doc("collectd-interface.md")),
    ({"type": "collectd/memory"}, get_metrics_from_doc("collectd-memory.md")),
    ({"type": "collectd/nginx", "discoveryRule": 'container_image =~ "nginx" && private_port == 80'}, get_metrics_from_doc("collectd-nginx.md")),
    ({"type": "collectd/protocols"}, get_metrics_from_doc("collectd-protocols.md")),
    ({"type": "collectd/signalfx-metadata", "procFSPath": "/hostfs/proc", "etcPath": "/hostfs/etc", "persistencePath": "/run"}, get_metrics_from_doc("collectd-signalfx-metadata.md", ignore=['cpu.utilization_per_core', 'disk.summary_utilization', 'disk.utilization', 'disk_ops.total'])),
    ({"type": "collectd/uptime"}, get_metrics_from_doc("collectd-uptime.md")),
    ({"type": "collectd/vmem"}, get_metrics_from_doc("collectd-vmem.md")),
    ({"type": "docker-container-stats"}, get_metrics_from_doc("docker-container-stats.md", ignore=["memory.stats.swap"])),
    ({"type": "internal-metrics"}, get_metrics_from_doc("internal-metrics.md")),
    ({"type": "kubelet-stats", "kubeletAPI": {"skipVerify": True, "authType": "serviceAccount"}}, get_metrics_from_doc("kubelet-stats.md")),
    #({"type": "kubernetes-cluster", "kubernetesAPI": {"skipVerify": True, "authType": "serviceAccount"}}, get_metrics_from_doc("kubernetes-cluster.md")),
    ({"type": "kubernetes-cluster", "kubernetesAPI": {"authType": "serviceAccount"}}, get_metrics_from_doc("kubernetes-cluster.md")),
    ({"type": "kubernetes-volumes", "kubeletAPI": {"skipVerify": True, "authType": "serviceAccount"}}, ['kubernetes.volume_available_bytes', 'kubernetes.volume_capacity_bytes']),
]

@pytest.mark.parametrize("monitor,expected_metrics", MONITORS_TO_TEST, ids=[m[0]["type"] for m in MONITORS_TO_TEST])
def test_metrics(monitor, expected_metrics, minikube, k8s_test_timeout):
    if monitor["type"] == 'collectd/nginx' and minikube.agent.observer in ['docker', 'host']:
        pytest.skip("skipping monitor '%s' test for observer '%s'" % (monitor["type"], minikube.agent.observer))
    if len(expected_metrics) == 0:
        pytest.skip("expected metrics is empty; skipping test")
    print("\nCollected %d metric(s) to test for %s." % (len(expected_metrics), monitor["type"]))
    metrics_not_found = check_for_metrics(minikube.agent.backend, expected_metrics, k8s_test_timeout)
    assert len(metrics_not_found) == 0, "timed out waiting for metric(s): %s\n\n%s\n\n" % (metrics_not_found, get_all_logs(minikube))

def test_dims(expected_dims, minikube, k8s_test_timeout):
    if len(expected_dims) == 0:
        pytest.skip("expected dimensions is empty; skipping test")
    print("\nCollected %d dimension(s) to test." % len(expected_dims))
    dims_not_found = check_for_dims(minikube.agent.backend, expected_dims, k8s_test_timeout)
    assert len(dims_not_found) == 0, "timed out waiting for dimension(s): %s\n\n%s\n\n" % (dims_not_found, get_all_logs(minikube))


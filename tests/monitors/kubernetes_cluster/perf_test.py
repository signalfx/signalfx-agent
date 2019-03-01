import time
from functools import partial as p
from textwrap import dedent

import pytest
from kubernetes import client
from tests.helpers.assertions import has_datapoint, has_dim_prop
from tests.helpers.kubernetes.fakeapiserver import fake_k8s_api_server
from tests.helpers.util import ensure_always, run_agent, wait_for

pytestmark = [pytest.mark.kubernetes_cluster, pytest.mark.perf_test]


def test_large_kubernetes_clusters():
    pod_count = 5000
    with fake_k8s_api_server(print_logs=True) as [fake_k8s_client, k8s_envvars]:
        names = []
        uids = []

        v1_client = client.CoreV1Api(fake_k8s_client)
        for i in range(0, pod_count):
            name = f"pod-{i}"
            names.append(name)

            uid = f"abcdefg{i}"
            uids.append(uid)

            v1_client.create_namespaced_pod(
                body={
                    "apiVersion": "v1",
                    "kind": "Pod",
                    "metadata": {"name": name, "uid": uid, "namespace": "default", "labels": {"app": "my-app"}},
                    "spec": {},
                },
                namespace="default",
            )

        with run_agent(
            dedent(
                f"""
          writer:
            maxRequests: 100
            propertiesMaxRequests: 100
            propertiesHistorySize: 10000
          monitors:
           - type: internal-metrics
             intervalSeconds: 1
           - type: kubernetes-cluster
             alwaysClusterReporter: true
             intervalSeconds: 10
             kubernetesAPI:
                skipVerify: true
                authType: none
        """
            ),
            profile=True,
            debug=False,
            extra_env=k8s_envvars,
        ) as [backend, _, _, pprof_client]:
            assert wait_for(p(has_datapoint, backend, dimensions={"kubernetes_pod_name": "pod-0"}))
            assert wait_for(p(has_datapoint, backend, dimensions={"kubernetes_pod_name": "pod-4999"}))

            def has_all_pod_datapoints():
                for name in names:
                    if not has_datapoint(backend, dimensions={"kubernetes_pod_name": name}):
                        return False
                return True

            def has_all_pod_properties():
                for uid in uids:
                    if not has_dim_prop(
                        backend, dim_name="kubernetes_pod_uid", dim_value=uid, prop_name="app", prop_value="my-app"
                    ):
                        return False
                return True

            assert wait_for(has_all_pod_datapoints, interval_seconds=2)
            assert wait_for(has_all_pod_properties, interval_seconds=2)

            for name in names:
                v1_client.delete_namespaced_pod(name=name, namespace="default", body={})

            time.sleep(10)
            backend.reset_datapoints()

            def has_no_pod_datapoints():
                for name in names:
                    if has_datapoint(backend, dimensions={"kubernetes_pod_name": name}):
                        return False
                return True

            assert ensure_always(has_no_pod_datapoints, interval_seconds=2)

            pprof_client.save_goroutines()
            assert (
                backend.datapoints_by_metric["sfxagent.go_num_goroutine"][-1].value.intValue < 100
            ), "too many goroutines"

            assert (
                backend.datapoints_by_metric["sfxagent.go_heap_alloc"][-1].value.intValue < 200 * 1024 * 1024
            ), "too much memory used"

import time
from functools import partial as p
from textwrap import dedent

import pytest
from kubernetes import client
from tests.helpers.assertions import has_datapoint, has_dim_prop, has_dim_tag
from tests.helpers.kubernetes.fakeapiserver import fake_k8s_api_server
from tests.helpers.util import ensure_always, run_agent, wait_for

pytestmark = [pytest.mark.kubernetes_cluster, pytest.mark.perf_test]


def test_large_kubernetes_clusters():
    pod_count = 5000
    with fake_k8s_api_server(print_logs=True) as [fake_k8s_client, k8s_envvars]:
        pod_names = []
        uids = []

        v1_client = client.CoreV1Api(fake_k8s_client)
        for i in range(0, pod_count):
            name = f"pod-{i}"
            pod_names.append(name)

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
                for name in pod_names:
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

            for name in pod_names:
                v1_client.delete_namespaced_pod(name=name, namespace="default", body={})

            time.sleep(10)
            backend.reset_datapoints()

            def has_no_pod_datapoints():
                for name in pod_names:
                    if has_datapoint(backend, dimensions={"kubernetes_pod_name": name}):
                        return False
                return True

            assert ensure_always(has_no_pod_datapoints, interval_seconds=2)

            pprof_client.save_goroutines()
            assert wait_for(
                lambda: backend.datapoints_by_metric["sfxagent.go_num_goroutine"][-1].value.intValue < 100,
                timeout_seconds=7,
            ), "too many goroutines"

            assert (
                backend.datapoints_by_metric["sfxagent.go_heap_alloc"][-1].value.intValue < 200 * 1024 * 1024
            ), "too much memory used"


# pylint: disable=too-many-locals
def test_large_kubernetes_cluster_service_tags():
    pod_count = 5000
    service_count = 25
    with fake_k8s_api_server(print_logs=True) as [fake_k8s_client, k8s_envvars]:
        pod_names = []
        uids = []
        service_names = []

        v1_client = client.CoreV1Api(fake_k8s_client)
        ## create pods
        for i in range(0, pod_count):
            name = f"pod-{i}"
            pod_names.append(name)

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
        ## create services
        for i in range(0, service_count):
            service_name = f"service-{i}"
            service_names.append(service_name)
            v1_client.create_namespaced_service(
                body={
                    "apiVersion": "v1",
                    "kind": "Service",
                    "metadata": {"name": service_name, "uid": f"serviceUID{i}", "namespace": "default"},
                    "spec": {"selector": {"app": "my-app"}, "type": "LoadBalancer"},
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

            # assert wait_for(missing_service_tags, interval_seconds=2)

            def has_all_service_tags():
                for uid in uids:
                    for s_name in service_names:
                        if not has_dim_tag(
                            backend,
                            dim_name="kubernetes_pod_uid",
                            dim_value=uid,
                            tag_value=f"kubernetes_service_{s_name}",
                        ):
                            return False
                return True

            def has_all_pod_datapoints():
                for name in pod_names:
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
            assert wait_for(has_all_service_tags, interval_seconds=2)

            ## delete all services and make sure no pods have service tags
            for s_name in service_names:
                v1_client.delete_namespaced_service(name=s_name, namespace="default", body={})

            def missing_service_tags():
                for uid in uids:
                    for s_name in service_names:
                        if has_dim_tag(
                            backend,
                            dim_name="kubernetes_pod_uid",
                            dim_value=uid,
                            tag_value=f"kubernetes_service_{s_name}",
                        ):
                            return False
                return True

            assert wait_for(missing_service_tags, interval_seconds=2, timeout_seconds=60)

            pprof_client.save_goroutines()
            assert wait_for(
                lambda: backend.datapoints_by_metric["sfxagent.go_num_goroutine"][-1].value.intValue < 100,
                timeout_seconds=5,
            ), "too many goroutines"

            assert (
                backend.datapoints_by_metric["sfxagent.go_heap_alloc"][-1].value.intValue < 200 * 1024 * 1024
            ), "too much memory used"

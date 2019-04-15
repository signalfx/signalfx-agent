# pylint: disable=too-many-locals
import time
from functools import partial as p
from textwrap import dedent

import pytest
from kubernetes import client
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, has_dim_prop, has_dim_tag
from tests.helpers.kubernetes.fakeapiserver import fake_k8s_api_server
from tests.helpers.util import ensure_always, wait_for

pytestmark = [pytest.mark.kubernetes_cluster, pytest.mark.perf_test]


def test_large_kubernetes_clusters():
    pod_count = 5000
    with fake_k8s_api_server(print_logs=False) as [fake_k8s_client, k8s_envvars]:
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

        with Agent.run(
            dedent(
                f"""
          writer:
            maxRequests: 100
            propertiesMaxRequests: 100
            propertiesHistorySize: 10000
          monitors:
           - type: kubernetes-cluster
             alwaysClusterReporter: true
             intervalSeconds: 10
             kubernetesAPI:
                skipVerify: true
                authType: none
        """
            ),
            profiling=True,
            debug=False,
            extra_env=k8s_envvars,
        ) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"kubernetes_pod_name": "pod-0"}))
            assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"kubernetes_pod_name": "pod-4999"}))

            def has_all_pod_datapoints():
                for name in pod_names:
                    if not has_datapoint(agent.fake_services, dimensions={"kubernetes_pod_name": name}):
                        return False
                return True

            def has_all_pod_properties():
                for uid in uids:
                    if not has_dim_prop(
                        agent.fake_services,
                        dim_name="kubernetes_pod_uid",
                        dim_value=uid,
                        prop_name="app",
                        prop_value="my-app",
                    ):
                        return False
                return True

            assert wait_for(has_all_pod_datapoints, interval_seconds=2)
            assert wait_for(has_all_pod_properties, interval_seconds=2)

            assert (
                agent.internal_metrics_client.get()["sfxagent.dim_updates_completed"] == 5000
            ), "Got wrong number of dimension updates"

            for name in pod_names:
                v1_client.delete_namespaced_pod(name=name, namespace="default", body={})

            time.sleep(10)
            agent.fake_services.reset_datapoints()

            def has_no_pod_datapoints():
                for name in pod_names:
                    if has_datapoint(agent.fake_services, dimensions={"kubernetes_pod_name": name}):
                        return False
                return True

            assert ensure_always(has_no_pod_datapoints, interval_seconds=2)

            agent.pprof_client.assert_goroutine_count_under(200)
            agent.pprof_client.assert_heap_alloc_under(200 * 1024 * 1024)


# pylint: disable=too-many-locals
def test_service_tag_sync():
    pod_count = 5000
    service_count = 25
    with fake_k8s_api_server(print_logs=False) as [fake_k8s_client, k8s_envvars]:
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

        with Agent.run(
            dedent(
                f"""
          writer:
            maxRequests: 100
            propertiesMaxRequests: 10
            propertiesHistorySize: 10000
            propertiesSendDelaySeconds: 5
          monitors:
           - type: kubernetes-cluster
             alwaysClusterReporter: true
             intervalSeconds: 10
             kubernetesAPI:
                skipVerify: true
                authType: none
        """
            ),
            profiling=True,
            debug=False,
            extra_env=k8s_envvars,
        ) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"kubernetes_pod_name": "pod-0"}))
            assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"kubernetes_pod_name": "pod-4999"}))

            # assert wait_for(missing_service_tags, interval_seconds=2)

            def has_all_service_tags():
                for uid in uids:
                    for s_name in service_names:
                        if not has_dim_tag(
                            agent.fake_services,
                            dim_name="kubernetes_pod_uid",
                            dim_value=uid,
                            tag_value=f"kubernetes_service_{s_name}",
                        ):
                            return False
                return True

            def has_all_pod_datapoints():
                for name in pod_names:
                    if not has_datapoint(agent.fake_services, dimensions={"kubernetes_pod_name": name}):
                        return False
                return True

            def has_all_pod_properties():
                for uid in uids:
                    if not has_dim_prop(
                        agent.fake_services,
                        dim_name="kubernetes_pod_uid",
                        dim_value=uid,
                        prop_name="app",
                        prop_value="my-app",
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
                            agent.fake_services,
                            dim_name="kubernetes_pod_uid",
                            dim_value=uid,
                            tag_value=f"kubernetes_service_{s_name}",
                        ):
                            return False
                return True

            assert wait_for(missing_service_tags, interval_seconds=2, timeout_seconds=60)

            agent.pprof_client.assert_goroutine_count_under(150)
            agent.pprof_client.assert_heap_alloc_under(200 * 1024 * 1024)


# pylint: disable=too-many-locals
def test_large_k8s_cluster_deployment_prop():
    """
    Creates 50 replica sets with 100 pods per replica set.
    Check that the deployment name is being synced to
    kubernetes_pod_uid, which is taken off the replica set's
    owner references.
    """
    dp_count = 50
    pods_per_dp = 100
    with fake_k8s_api_server(print_logs=False) as [fake_k8s_client, k8s_envvars]:
        v1_client = client.CoreV1Api(fake_k8s_client)
        v1beta1_client = client.ExtensionsV1beta1Api(fake_k8s_client)
        replica_sets = {}
        for i in range(0, dp_count):
            dp_name = f"dp-{i}"
            dp_uid = f"dpuid{i}"
            rs_name = dp_name + "-replicaset"
            rs_uid = dp_uid + "-rs"
            replica_sets[rs_uid] = {
                "dp_name": dp_name,
                "dp_uid": dp_uid,
                "rs_name": rs_name,
                "rs_uid": rs_uid,
                "pod_uids": [],
                "pod_names": [],
            }

            v1beta1_client.create_namespaced_replica_set(
                body={
                    "apiVersion": "extensions/v1beta1",
                    "kind": "ReplicaSet",
                    "metadata": {
                        "name": rs_name,
                        "uid": rs_uid,
                        "namespace": "default",
                        "ownerReferences": [{"kind": "Deployment", "name": dp_name, "uid": dp_uid}],
                    },
                    "spec": {},
                    "status": {},
                },
                namespace="default",
            )

            for j in range(0, pods_per_dp):
                pod_name = f"pod-{rs_name}-{j}"
                pod_uid = f"abcdef{i}-{j}"
                replica_sets[rs_uid]["pod_uids"].append(pod_uid)
                replica_sets[rs_uid]["pod_names"].append(pod_name)
                v1_client.create_namespaced_pod(
                    body={
                        "apiVersion": "v1",
                        "kind": "Pod",
                        "metadata": {
                            "name": pod_name,
                            "uid": pod_uid,
                            "namespace": "default",
                            "labels": {"app": "my-app"},
                            "ownerReferences": [{"kind": "ReplicaSet", "name": rs_name, "uid": rs_uid}],
                        },
                        "spec": {},
                    },
                    namespace="default",
                )
        with Agent.run(
            dedent(
                f"""
          writer:
            maxRequests: 100
            propertiesMaxRequests: 100
            propertiesHistorySize: 10000
          monitors:
           - type: kubernetes-cluster
             alwaysClusterReporter: true
             intervalSeconds: 10
             kubernetesAPI:
                skipVerify: true
                authType: none
        """
            ),
            profiling=True,
            debug=False,
            extra_env=k8s_envvars,
        ) as agent:
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"kubernetes_pod_name": "pod-dp-0-replicaset-0"})
            )
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"kubernetes_pod_name": "pod-dp-49-replicaset-99"})
            )

            ## get heap usage with 5k pods
            heap_profile_baseline = agent.pprof_client.get_heap_profile()

            def has_all_deployment_props():
                for _, replica_set in replica_sets.items():
                    for pod_uid in replica_set["pod_uids"]:
                        if not has_dim_prop(
                            agent.fake_services,
                            dim_name="kubernetes_pod_uid",
                            dim_value=pod_uid,
                            prop_name="deployment",
                            prop_value=replica_set["dp_name"],
                        ):
                            return False
                        if not has_dim_prop(
                            agent.fake_services,
                            dim_name="kubernetes_pod_uid",
                            dim_value=pod_uid,
                            prop_name="deployment_uid",
                            prop_value=replica_set["dp_uid"],
                        ):
                            return False
                    return True

            def has_all_replica_set_props():
                for _, replica_set in replica_sets.items():
                    for pod_uid in replica_set["pod_uids"]:
                        if not has_dim_prop(
                            agent.fake_services,
                            dim_name="kubernetes_pod_uid",
                            dim_value=pod_uid,
                            prop_name="replicaSet",
                            prop_value=replica_set["rs_name"],
                        ):
                            return False
                        if not has_dim_prop(
                            agent.fake_services,
                            dim_name="kubernetes_pod_uid",
                            dim_value=pod_uid,
                            prop_name="replicaSet_uid",
                            prop_value=replica_set["rs_uid"],
                        ):
                            return False
                    return True

            assert wait_for(has_all_deployment_props, interval_seconds=2, timeout_seconds=60)
            assert wait_for(has_all_replica_set_props, interval_seconds=2)

            assert (
                agent.internal_metrics_client.get()["sfxagent.dim_updates_completed"] == 5050
            ), "Got wrong number of dimension updates"

            for _, replica_set in replica_sets.items():
                v1beta1_client.delete_namespaced_replica_set(name=replica_set["rs_name"], namespace="default", body={})
                for pod_name in replica_set["pod_names"]:
                    v1_client.delete_namespaced_pod(name=pod_name, namespace="default", body={})

            agent.pprof_client.assert_goroutine_count_under(200)
            agent.pprof_client.assert_heap_alloc_under(heap_profile_baseline.total * 1.2)

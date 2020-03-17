"""
Tests the service <--> container/pod correlation features in the agent.
"""
import json
import random
from functools import partial as p
from textwrap import dedent

import pytest
from kubernetes import client as kube_client
from tests.helpers.agent import Agent
from tests.helpers.assertions import (
    has_all_dim_props,
    has_datapoint,
    has_dim_set_prop,
    has_trace_span,
    tcp_port_open_locally,
)
from tests.helpers.kubernetes.utils import exec_pod_command
from tests.helpers.util import get_host_ip, get_stripped_container_id, run_service, wait_for
from tests.paths import TEST_SERVICES_DIR


# Make this a function so it returns a fresh copy on each call
def _test_trace():
    return [
        {
            "traceId": "0123456789abcdef",
            "name": "get",
            "id": "abcdef0123456789",
            "kind": "CLIENT",
            "timestamp": 1_538_406_065_536_000,
            "duration": 10000,
            "localEndpoint": {"serviceName": "myapp", "ipv4": "10.0.0.1"},
            "tags": {"env": "prod"},
        },
        {
            "traceId": "0123456789abcdef",
            "name": "fetch",
            "parentId": "abcdef0123456789",
            "id": "def0123456789abc",
            "kind": "SERVER",
            "timestamp": 1_538_406_068_536_000,
            "duration": 5000,
            "localEndpoint": {"serviceName": "myapp", "ipv4": "10.0.0.2"},
            "tags": {"env": "prod", "file": "test.pdf"},
        },
    ]


def test_docker_container_spans_get_container_id_tag():
    port = random.randint(5001, 20000)
    with Agent.run(
        dedent(
            f"""
        cluster: my-cluster
        writer:
          propertiesSendDelaySeconds: 1
        observers:
         - type: docker
        monitors:
          - type: docker-container-stats
          - type: trace-forwarder
            listenAddress: 0.0.0.0:{port}
    """
        )
    ) as agent:
        assert wait_for(p(tcp_port_open_locally, port)), "trace forwarder port never opened!"
        with run_service("curl", entrypoint=["tail", "-f", "/dev/null"]) as container:
            # This is purely to wait for the docker observer to have discovered
            # the container.
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"container_id": container.id})
            ), "Didn't get container datapoint"

            exit_code, out = container.exec_run(
                [
                    "curl",
                    f"http://{get_host_ip()}:{port}/v1/trace",
                    "-H",
                    "Content-Type: application/json",
                    "-d",
                    json.dumps(_test_trace()),
                ]
            )
            assert exit_code == 0, "Curl command failed: " + str(out)

            assert wait_for(
                p(has_trace_span, agent.fake_services, tags={"env": "prod", "container_id": container.id})
            ), "Didn't get span tag"

            assert wait_for(
                p(
                    has_all_dim_props,
                    agent.fake_services,
                    dim_name="container_id",
                    dim_value=container.id,
                    props={"service": "myapp", "cluster": "my-cluster"},
                )
            )

            assert wait_for(
                p(
                    has_dim_set_prop,
                    agent.fake_services,
                    dim_name="container_id",
                    dim_value=container.id,
                    prop_name="sf_services",
                    prop_values=["myapp"],
                )
            )


@pytest.mark.kubernetes
def test_k8s_pod_spans_get_pod_and_container_tags(k8s_cluster):
    port = random.randint(5001, 20000)
    config = f"""
        cluster: my-cluster
        writer:
          propertiesSendDelaySeconds: 1
        observers:
         - type: k8s-api
        monitors:
          - type: kubernetes-cluster
          - type: kubelet-stats
          - type: trace-forwarder
            listenAddress: 0.0.0.0:{port}
    """
    yamls = [TEST_SERVICES_DIR / "curl/curl-k8s.yaml"]
    with k8s_cluster.create_resources(yamls):
        with k8s_cluster.run_agent(agent_yaml=config) as agent:
            curl_pod = (
                kube_client.CoreV1Api()
                .list_namespaced_pod(k8s_cluster.test_namespace, watch=False, label_selector="app=curl-test")
                .items[0]
            )
            # This is to wait for the k8s-api observer to have discovered the
            # pod.
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"kubernetes_pod_uid": curl_pod.metadata.uid})
            ), "Didn't get pod datapoint"

            exec_pod_command(
                curl_pod.metadata.name,
                [
                    "curl",
                    f"http://{curl_pod.status.host_ip}:{port}/v1/trace",
                    "-H",
                    "Content-Type: application/json",
                    "-d",
                    json.dumps(_test_trace()),
                ],
                namespace=curl_pod.metadata.namespace,
                fail_hard=True,
            )
            container_id = get_stripped_container_id(curl_pod.status.container_statuses[0].container_id)

            assert wait_for(
                p(
                    has_trace_span,
                    agent.fake_services,
                    tags={"env": "prod", "container_id": container_id, "kubernetes_pod_uid": curl_pod.metadata.uid},
                )
            ), "Didn't get span tag with kubernetes_pod_uid added"

            assert wait_for(
                p(
                    has_all_dim_props,
                    agent.fake_services,
                    dim_name="kubernetes_pod_uid",
                    dim_value=curl_pod.metadata.uid,
                    props={"service": "myapp", "cluster": "my-cluster"},
                )
            )

            assert wait_for(
                p(
                    has_all_dim_props,
                    agent.fake_services,
                    dim_name="container_id",
                    dim_value=container_id,
                    props={"service": "myapp", "cluster": "my-cluster"},
                )
            )

            assert wait_for(
                p(
                    has_dim_set_prop,
                    agent.fake_services,
                    dim_name="kubernetes_pod_uid",
                    dim_value=curl_pod.metadata.uid,
                    prop_name="sf_services",
                    prop_values=["myapp"],
                )
            )

            assert wait_for(
                p(
                    has_dim_set_prop,
                    agent.fake_services,
                    dim_name="container_id",
                    dim_value=container_id,
                    prop_name="sf_services",
                    prop_values=["myapp"],
                )
            )

import os
import random
import shlex
import string
import subprocess
import time
from contextlib import contextmanager
from functools import partial as p
from pathlib import Path

import yaml
from kubernetes import client
from kubernetes import config as kconfig
from tests.helpers.fake_backend import start as start_fake_backend
from tests.helpers.formatting import print_dp_or_event
from tests.helpers.util import wait_for

from . import tunnel, utils
from .agent import Agent


class Cluster:
    def __init__(self, kube_config_path, kube_context=None, agent_image_name=None):
        self.kube_config_path = kube_config_path
        self.kube_context = kube_context
        self.agent_image_name = agent_image_name

        chars = string.ascii_lowercase + string.digits
        self.test_namespace = "pytest-" + "".join((random.choice(chars)) for x in range(8))

        print(f"Using test namespace: {self.test_namespace}")
        print(f"Using kube config file '{kube_config_path}' with context '{kube_context}'")
        print(f"Using agent image '{agent_image_name}'")

        kconfig.load_kube_config(config_file=kube_config_path, context=kube_context)

        api = client.CoreV1Api()

        self.container_runtimes = [
            node.status.node_info.container_runtime_version.split(":", 1)[0] for node in api.list_node().items
        ]
        assert self.container_runtimes, "failed to get container runtimes for cluster"

        utils.create_namespace(self.test_namespace)
        assert wait_for(p(utils.has_namespace, self.test_namespace))
        assert wait_for(p(utils.has_serviceaccount, "default", self.test_namespace))
        assert wait_for(lambda: api.list_namespaced_secret(self.test_namespace).items)

    def delete_test_namespace(self):
        """
        Cleans up the test namespace used in the cluser by deleting it.
        """
        client.CoreV1Api().delete_namespace(
            self.test_namespace, grace_period_seconds=0, body=client.V1DeleteOptions(propagation_policy="Background")
        )

    def exec_kubectl(self, command, namespace=None):
        """
        Runs kubectl with the given command in the given namespace.
        """
        if namespace is None:
            namespace = self.test_namespace
        args = ["kubectl", "-n", namespace]
        if self.kube_config_path:
            args += ["--kubeconfig", self.kube_config_path]
        if self.kube_context:
            args += ["--context", self.kube_context]

        args += shlex.split(command)

        proc = subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, encoding="utf-8", close_fds=False)
        assert proc.returncode == 0, f"{args}:\n{proc.stdout}"
        return proc.stdout

    def get_cluster_version(self):
        version_yaml = self.exec_kubectl("version --output=yaml")
        assert version_yaml, "failed to get kubectl version"
        cluster_version = yaml.safe_load(version_yaml).get("serverVersion").get("gitVersion")
        return cluster_version

    def wait_for_deployments(self, resources, timeout_seconds=utils.K8S_CREATE_TIMEOUT):
        for doc in filter(lambda d: d.kind == "Deployment", resources):
            name = doc.metadata.name
            namespace = doc.metadata.namespace
            print("Waiting for deployment %s to be ready ..." % name)
            try:
                start_time = time.time()
                assert wait_for(
                    p(utils.deployment_is_ready, name, namespace), timeout_seconds=timeout_seconds, interval_seconds=2
                ), 'timed out waiting for deployment "%s" to be ready!\n%s' % (
                    name,
                    utils.get_pod_logs(name, namespace),
                )
                print("Waited %d seconds" % (time.time() - start_time))
            finally:
                print(self.exec_kubectl(f"describe deployment {name}", namespace=namespace))
                for pod in utils.get_all_pods(namespace):
                    print(self.exec_kubectl(f"describe pod {pod.metadata.name}", namespace=namespace))

    @contextmanager
    def create_resources(self, yamls, timeout=utils.K8S_CREATE_TIMEOUT):
        resources = []
        for yaml_resource in yamls:
            yaml_bytes = yaml_resource

            if os.path.isfile(yaml_resource):
                yaml_bytes = Path(yaml_resource).read_bytes()

            for doc in yaml.safe_load_all(yaml_bytes):
                kind = doc["kind"]
                name = doc["metadata"]["name"]
                namespace = doc["metadata"].setdefault("namespace", self.test_namespace)
                api_client = utils.api_client_from_version(doc["apiVersion"])

                print(f"Creating {kind}/{name}")
                new_resource = utils.create_resource(doc, api_client, namespace=namespace, timeout=timeout)
                resources.append(new_resource)

        self.wait_for_deployments(resources)

        try:
            yield resources
        finally:
            for res in resources:
                kind = res.kind
                name = res.metadata.name
                print('Deleting %s "%s" ...' % (kind, name))
                namespace = res.metadata.namespace
                api_client = utils.api_client_from_version(res.api_version)
                utils.delete_resource(name, kind, api_client, namespace=namespace)

    @contextmanager
    def run_agent(self, agent_yaml):
        with start_fake_backend() as backend:
            with self.run_tunnels(backend) as pod_ip:
                agent = Agent(
                    agent_image_name=self.agent_image_name,
                    cluster=self,
                    namespace=self.test_namespace,
                    fake_services=backend,
                    fake_services_pod_ip=pod_ip,
                )
                with agent.deploy(agent_yaml):
                    try:
                        yield agent
                    finally:
                        print("\nDatapoints received:")
                        for dp in backend.datapoints or []:
                            print_dp_or_event(dp)

                        print("\nEvents received:")
                        for event in backend.events or []:
                            print_dp_or_event(event)
                        print(f"\nDimensions set: {backend.dims}")
                        print("\nTrace spans received:")
                        for span in backend.spans or []:
                            print(span)

    @contextmanager
    def run_tunnels(self, fake_services):
        with tunnel.deploy_fake_backend_proxy_pod(self.test_namespace) as pod_ip:
            terminate_ingest_tunnel, get_ingest_socat_output = tunnel.start_tunneling_fake_service(
                fake_services.ingest_host,
                fake_services.ingest_port,
                namespace=self.test_namespace,
                kube_config_path=self.kube_config_path,
                context=self.kube_context,
            )

            terminate_api_tunnel, get_api_socat_output = tunnel.start_tunneling_fake_service(
                fake_services.api_host,
                fake_services.api_port,
                namespace=self.test_namespace,
                kube_config_path=self.kube_config_path,
                context=self.kube_context,
            )

            try:
                yield pod_ip
            finally:
                terminate_ingest_tunnel()
                print("Ingest tunnel output: " + get_ingest_socat_output())
                terminate_api_tunnel()
                print("API tunnel output: " + get_api_socat_output())

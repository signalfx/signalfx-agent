import os
import tempfile
import time
from contextlib import contextmanager
from functools import partial as p

import docker
import semver
import yaml
from kubernetes import config as kube_config

from helpers.assertions import container_cmd_exit_0
from helpers.kubernetes.agent import Agent
from helpers.kubernetes.utils import (
    api_client_from_version,
    container_is_running,
    create_resource,
    delete_resource,
    get_all_logs,
    get_free_port,
    has_docker_image,
    has_resource,
    wait_for_deployment,
)
from helpers.util import container_ip, get_docker_client, wait_for

MINIKUBE_VERSION = os.environ.get("MINIKUBE_VERSION")
MINIKUBE_LOCALKUBE_VERSION = "v0.28.2"
MINIKUBE_KUBEADM_VERSION = "v0.30.0"
TEST_SERVICES_DIR = os.path.join(os.path.dirname(os.path.realpath(__file__)), "../../../test-services")


class Minikube:  # pylint: disable=too-many-instance-attributes
    def __init__(self):
        self.bootstrapper = None
        self.container = None
        self.client = None
        self.version = None
        self.k8s_version = None
        self.name = None
        self.host_client = get_docker_client()
        self.yamls = []
        self.agent = Agent()
        self.cluster_name = "minikube"
        self.kubeconfig = None
        self.namespace = "default"
        self.worker_id = "master"
        self.registry_port = None

    def get_client(self):
        if self.container:
            self.container.reload()
            self.client = docker.DockerClient(base_url="tcp://%s:2375" % container_ip(self.container), version="auto")

        return self.client

    def load_kubeconfig(self, kubeconfig_path="/kubeconfig", timeout=300):
        with tempfile.NamedTemporaryFile(dir="/tmp/scratch") as fd:
            kubeconfig = fd.name
            assert wait_for(
                p(container_cmd_exit_0, self.container, "test -f %s" % kubeconfig_path),
                timeout_seconds=timeout,
                interval_seconds=2,
            ), ("timed out waiting for the minikube cluster to be ready!\n\n%s\n\n" % self.get_logs())
            time.sleep(2)
            exit_code, output = self.container.exec_run("cp -f %s %s" % (kubeconfig_path, kubeconfig))
            assert exit_code == 0, "failed to get %s from minikube!\n%s" % (kubeconfig_path, output.decode("utf-8"))
            self.kubeconfig = kubeconfig
            kube_config.load_kube_config(config_file=self.kubeconfig)

    def get_bootstrapper(self):
        code, _ = self.container.exec_run("which localkube")
        if code == 0:
            self.bootstrapper = "localkube"
        else:
            code, _ = self.container.exec_run("which kubeadm")
            if code == 0:
                self.bootstrapper = "kubeadm"
        return self.bootstrapper

    def connect(self, name, timeout, version=None):
        print("\nConnecting to %s container ..." % name)
        assert wait_for(p(container_is_running, self.host_client, name), timeout_seconds=timeout, interval_seconds=2), (
            "timed out waiting for container %s!" % name
        )
        self.container = self.host_client.containers.get(name)
        self.load_kubeconfig(timeout=timeout)
        self.client = self.get_client()
        self.get_bootstrapper()
        self.name = name
        self.k8s_version = version

    def deploy(self, version, timeout, options=None):
        if options is None:
            options = {}
        self.registry_port = get_free_port()
        if container_is_running(self.host_client, "minikube"):
            self.host_client.containers.get("minikube").remove(force=True, v=True)
        self.k8s_version = version
        if self.k8s_version[0] != "v":
            self.k8s_version = "v" + self.k8s_version
        if not options:
            options = {
                "name": "minikube",
                "privileged": True,
                "environment": {"K8S_VERSION": self.k8s_version, "TIMEOUT": str(timeout)},
                "ports": {
                    "8080/tcp": None,
                    "8443/tcp": None,
                    "2375/tcp": None,
                    "%d/tcp" % self.registry_port: self.registry_port,
                },
                "volumes": {"/tmp/scratch": {"bind": "/tmp/scratch", "mode": "rw"}},
            }
        if MINIKUBE_VERSION:
            self.version = MINIKUBE_VERSION
        elif semver.match(self.k8s_version.lstrip("v"), ">=1.11.0"):
            self.version = MINIKUBE_KUBEADM_VERSION
        else:
            self.version = MINIKUBE_LOCALKUBE_VERSION
        if self.version == "latest" or semver.match(
            self.version.lstrip("v"), ">" + MINIKUBE_LOCALKUBE_VERSION.lstrip("v")
        ):
            options["command"] = "/lib/systemd/systemd"
        else:
            options["command"] = "sleep inf"
        print("\nDeploying minikube %s cluster ..." % self.k8s_version)
        image, _ = self.host_client.images.build(
            path=os.path.join(TEST_SERVICES_DIR, "minikube"),
            buildargs={"MINIKUBE_VERSION": self.version},
            tag="minikube:%s" % self.version,
            rm=True,
            forcerm=True,
        )
        self.container = self.host_client.containers.run(image.id, detach=True, **options)
        self.name = self.container.name
        self.container.exec_run("start-minikube.sh", detach=True)
        self.load_kubeconfig(timeout=timeout)
        self.client = self.get_client()
        self.get_bootstrapper()

    def start_registry(self):
        if not self.client:
            self.client = self.get_client()
        print("\nStarting registry container localhost:%d in minikube ..." % self.registry_port)
        self.client.containers.run(
            image="registry:latest",
            name="registry",
            detach=True,
            environment={"REGISTRY_HTTP_ADDR": "0.0.0.0:%d" % self.registry_port},
            ports={"%d/tcp" % self.registry_port: self.registry_port},
        )

    def build_image(self, dockerfile_dir, build_opts=None):
        if build_opts is None:
            build_opts = {}
        if not self.client:
            self.get_client()
        self.client.images.build(path=dockerfile_dir, rm=True, forcerm=True, **build_opts)

    @contextmanager
    def deploy_k8s_yamls(self, yamls=None, namespace=None, timeout=180):
        if yamls is None:
            yamls = []
        self.yamls = []
        for yaml_file in yamls:
            assert os.path.isfile(yaml_file), '"%s" not found!' % yaml_file
            docs = []
            with open(yaml_file, "r") as fd:
                docs = yaml.load_all(fd.read())

            for doc in docs:
                kind = doc["kind"]
                name = doc["metadata"]["name"]
                api_version = doc["apiVersion"]
                api_client = api_client_from_version(api_version)

                if not doc.get("metadata", {}).get("namespace"):
                    if "metadata" not in doc:
                        doc["metadata"] = {}
                    doc["metadata"]["namespace"] = namespace

                if has_resource(name, kind, api_client, namespace):
                    print('Deleting %s "%s" ...' % (kind, name))
                    delete_resource(name, kind, api_client, namespace=namespace)

                print("Creating %s from %s ..." % (kind, yaml_file))
                create_resource(doc, api_client, namespace=namespace, timeout=timeout)
                self.yamls.append(doc)

        for doc in filter(lambda d: d["kind"] == "Deployment", self.yamls):
            print("Waiting for ports to open on deployment %s" % doc["metadata"]["name"])
            wait_for_deployment(doc, self.container, timeout)

        try:
            yield
        finally:
            for res in self.yamls:
                print('Deleting %s "%s" ...' % (kind, name))
                kind = res["kind"]
                api_version = res["apiVersion"]
                api_client = api_client_from_version(api_version)
                delete_resource(name, kind, api_client, namespace=namespace)
            self.yamls = []

    def pull_agent_image(self, name, tag, image_id=None):
        if image_id and has_docker_image(self.client, image_id):
            return self.client.images.get(image_id)

        if has_docker_image(self.client, name, tag):
            return self.client.images.get(name + ":" + tag)

        return self.client.images.pull(name, tag=tag)

    @contextmanager
    def deploy_agent(
        self,
        configmap_path,
        daemonset_path,
        serviceaccount_path,
        clusterrole_path,
        clusterrolebinding_path,
        observer=None,
        monitors=None,
        cluster_name="minikube",
        backend=None,
        image_name=None,
        image_tag=None,
        namespace="default",
    ):  # pylint: disable=too-many-arguments
        if monitors is None:
            monitors = []
        self.agent.deploy(
            self.client,
            configmap_path,
            daemonset_path,
            serviceaccount_path,
            clusterrole_path,
            clusterrolebinding_path,
            observer,
            monitors,
            cluster_name=cluster_name,
            backend=backend,
            image_name=image_name,
            image_tag=image_tag,
            namespace=namespace,
        )
        try:
            yield self.agent
            print("\nAgent status:\n%s\n" % self.agent.get_status())
            print("\nAgent container logs:\n%s\n" % self.agent.get_container_logs())
        except Exception:
            print("\n%s\n" % get_all_logs(self))
            raise
        finally:
            self.agent.delete()
            self.agent = Agent()

    def get_container_logs(self):
        try:
            return self.container.logs().decode("utf-8").strip()
        except Exception as e:  # pylint: disable=broad-except
            return "Failed to get minikube container logs!\n%s" % str(e)

    def get_localkube_logs(self):
        try:
            exit_code, _ = self.container.exec_run("test -f /var/lib/localkube/localkube.err")
            if exit_code == 0:
                _, output = self.container.exec_run("cat /var/lib/localkube/localkube.err")
                return output.decode("utf-8").strip()
        except Exception as e:  # pylint: disable=broad-except
            return "Failed to get localkube logs from minikube!\n%s" % str(e)
        return None

    def get_logs(self):
        if self.container and self.bootstrapper:
            _, start_minikube_output = self.container.exec_run("cat /var/log/start-minikube.log")
            if self.bootstrapper == "localkube":
                return "/var/log/start-minikube.log:\n%s\n\n/var/lib/localkube/localkube.err:\n%s" % (
                    start_minikube_output.decode("utf-8").strip(),
                    self.get_localkube_logs(),
                )
            if self.bootstrapper == "kubeadm":
                _, minikube_logs = self.container.exec_run("minikube logs")
                return "/var/log/start-minikube.log:\n%s\n\nminikube logs:\n%s" % (
                    start_minikube_output.decode("utf-8").strip(),
                    minikube_logs.decode("utf-8").strip(),
                )
        return ""

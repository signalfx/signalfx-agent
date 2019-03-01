import json
import os
import re
import socket
import tempfile
import time
import urllib
from contextlib import contextmanager
from functools import partial as p

import docker
import requests
import semver
import yaml
from kubernetes import config as kube_config

import tests.helpers.kubernetes.utils as k8s
from tests.helpers.assertions import container_cmd_exit_0, tcp_socket_open
from tests.helpers.formatting import print_dp_or_event
from tests.helpers.kubernetes.agent import Agent
from tests.helpers.util import (
    container_ip,
    fake_backend,
    get_docker_client,
    get_host_ip,
    retry,
    wait_for,
    TEST_SERVICES_DIR,
)
from tests.packaging.common import get_container_file_content

MINIKUBE_CONTAINER_NAME = "minikube"
MINIKUBE_IMAGE_NAME = "minikube"
MINIKUBE_VERSION = os.environ.get("MINIKUBE_VERSION")
MINIKUBE_LOCALKUBE_VERSION = "0.28.2"
MINIKUBE_KUBEADM_VERSION = "0.34.1"
MINIKUBE_IMAGE_TIMEOUT = int(os.environ.get("MINIKUBE_IMAGE_TIMEOUT", 300))
MINIKUBE_KUBECONFIG_PATH = "/kubeconfig"
K8S_API_PORT = 8443
K8S_RELEASE_URL = "https://storage.googleapis.com/kubernetes-release/release/stable.txt"
K8S_MIN_VERSION = "1.7.0"
K8S_MIN_KUBEADM_VERSION = "1.11.0"


def get_free_port():
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("", 0))
        return sock.getsockname()[1]


def get_latest_k8s_version():
    version = None
    with urllib.request.urlopen(K8S_RELEASE_URL) as resp:
        version = resp.read().decode("utf-8").strip()
    assert version, "Failed to get latest K8S version from %s" % K8S_RELEASE_URL
    return version.lstrip("v")


def check_k8s_version(k8s_version):
    assert k8s_version, "K8S version not defined"
    k8s_latest_version = retry(get_latest_k8s_version, urllib.error.URLError)
    if k8s_version.lower() == "latest":
        k8s_version = k8s_latest_version
    k8s_version = k8s_version.lstrip("v")
    assert re.match(r"^\d+\.\d+\.\d+$", k8s_version), "Invalid K8S version '%s'" % k8s_version
    assert semver.match(k8s_version, ">=" + K8S_MIN_VERSION), "K8S version %s not supported" % k8s_version
    assert semver.match(k8s_version, "<=" + k8s_latest_version), "K8S version %s not supported" % k8s_version
    return "v" + k8s_version


def has_docker_image(client, name, tag=None):
    if tag:
        name = name + ":" + tag
    return client.images.list(name=name)


class Minikube:  # pylint: disable=too-many-instance-attributes
    def __init__(self):
        self.agent = Agent()
        self.client = None
        self.cluster_name = None
        self.container = None
        self.container_ip = None
        self.container_name = None
        self.host_client = get_docker_client()
        self.image_tag = None
        self.k8s_version = None
        self.kubeconfig = None
        self.registry_port = 5000
        self.resources = []
        self.version = None

    def get_version(self, k8s_version):
        if MINIKUBE_VERSION:
            self.version = MINIKUBE_VERSION
        elif k8s_version.lower() == "latest" or semver.match(k8s_version.lstrip("v"), ">=" + K8S_MIN_KUBEADM_VERSION):
            self.version = MINIKUBE_KUBEADM_VERSION
        else:
            self.version = MINIKUBE_LOCALKUBE_VERSION
        if self.version:
            self.version = "v" + self.version.lstrip("v")
            self.image_tag = MINIKUBE_IMAGE_NAME + ":" + self.version
        return self.version

    def is_running(self):
        if not self.container:
            filters = {"name": self.container_name, "status": "running"}
            if self.host_client and self.container_name and self.host_client.containers.list(filters=filters):
                self.container = self.host_client.containers.get(self.container_name)
        if self.container:
            self.container.reload()
            self.container_ip = container_ip(self.container)
            return self.container.status == "running" and self.container_ip
        return False

    def is_ready(self):
        def kubeconfig_exists():
            try:
                return container_cmd_exit_0(self.container, "test -f %s" % MINIKUBE_KUBECONFIG_PATH)
            except requests.exceptions.RequestException as e:
                print("requests.exceptions.RequestException:\n%s" % str(e))
                return False

        return self.is_running() and tcp_socket_open(self.container_ip, K8S_API_PORT) and kubeconfig_exists()

    def exec_kubectl(self, command, namespace=None):
        if self.container:
            command = "kubectl %s" % command
            if namespace:
                command += " -n %s" % namespace
            return self.container.exec_run(command).output.decode("utf-8")
        return ""

    def get_cluster_version(self):
        version_yaml = self.exec_kubectl("version --output=yaml")
        assert version_yaml, "failed to get kubectl version"
        cluster_version = yaml.load(version_yaml).get("serverVersion").get("gitVersion")
        return check_k8s_version(cluster_version)

    def get_client(self):
        if not self.client:
            assert wait_for(self.is_running, timeout_seconds=30, interval_seconds=2), (
                "timed out waiting for %s container" % self.container_name
            )
            assert wait_for(p(tcp_socket_open, self.container_ip, 2375), timeout_seconds=30, interval_seconds=2), (
                "timed out waiting for docker engine in %s container!" % self.container_name
            )
            self.client = docker.DockerClient(base_url="tcp://%s:2375" % self.container_ip, version="auto")
        return self.client

    def get_logs(self):
        if self.is_running():
            return "/var/log/start-minikube.log:\n%s" % get_container_file_content(
                self.container, "/var/log/start-minikube.log"
            )
        return "%s container is not running" % self.container_name

    def connect_to_cluster(self, timeout=300):
        print("Waiting for minikube cluster to be ready ...")
        start_time = time.time()
        assert wait_for(self.is_ready, timeout_seconds=timeout, interval_seconds=2), (
            "timed out waiting for minikube cluster to be ready!\n%s" % self.get_logs()
        )
        print("Waited %d seconds" % (time.time() - start_time))
        time.sleep(2)
        if self.k8s_version:
            cluster_version = self.get_cluster_version()
            assert self.k8s_version.lstrip("v") == cluster_version.lstrip("v"), (
                "desired K8S version (%s) does not match actual cluster version (%s):\n%s"
                % (self.k8s_version, cluster_version, self.get_logs())
            )
        else:
            self.k8s_version = self.get_cluster_version()
        content = get_container_file_content(self.container, MINIKUBE_KUBECONFIG_PATH)
        self.kubeconfig = yaml.load(content)
        current_context = self.kubeconfig.get("current-context")
        for context in self.kubeconfig.get("contexts"):
            if context.get("name") == current_context:
                self.cluster_name = context.get("context").get("cluster")
                break
        assert self.cluster_name, "cluster not found in %s:\n%s" % (MINIKUBE_KUBECONFIG_PATH, content)
        with tempfile.NamedTemporaryFile(mode="w") as fd:
            fd.write(content)
            fd.flush()
            kube_config.load_kube_config(config_file=fd.name)
        self.get_client()
        print(self.exec_kubectl("version").strip())

    def connect(self, name=MINIKUBE_CONTAINER_NAME, k8s_version=None, timeout=300):
        self.container_name = name
        if k8s_version:
            assert self.get_version(k8s_version), "failed to get minikube version"
        if self.image_tag:
            start_time = time.time()
            print("Waiting for %s image to be built ..." % self.image_tag)
            assert wait_for(
                p(has_docker_image, self.host_client, self.image_tag),
                timeout_seconds=MINIKUBE_IMAGE_TIMEOUT,
                interval_seconds=2,
            ), ("timed out waiting for %s image to be built" % self.image_tag)
            print("Waited %d seconds" % (time.time() - start_time))
        print("\nConnecting to cluster in %s container ..." % self.container_name)
        self.connect_to_cluster(timeout)

    def deploy(self, k8s_version, timeout=300, options=None):
        self.k8s_version = check_k8s_version(k8s_version)
        assert self.get_version(k8s_version), "failed to get minikube version"
        if options is None:
            options = {}
        options.setdefault("name", MINIKUBE_CONTAINER_NAME)
        try:
            self.host_client.containers.get(options["name"]).remove(force=True, v=True)
        except docker.errors.NotFound:
            pass
        options.setdefault("privileged", True)
        options.setdefault(
            "environment",
            {"K8S_VERSION": self.k8s_version, "TIMEOUT": str(timeout), "KUBECONFIG_PATH": MINIKUBE_KUBECONFIG_PATH},
        )
        if tcp_socket_open("127.0.0.1", self.registry_port):
            self.registry_port = get_free_port()
        options.setdefault("ports", {"%d/tcp" % self.registry_port: self.registry_port})
        options.setdefault("detach", True)
        print("\nBuilding %s image ..." % self.image_tag)
        build_opts = dict(buildargs={"MINIKUBE_VERSION": self.version}, tag=self.image_tag)
        image_id = self.build_image("minikube", build_opts, "unix://var/run/docker.sock")
        print("\nDeploying minikube %s cluster ..." % self.k8s_version)
        self.container = self.host_client.containers.run(image_id, **options)
        self.container_name = self.container.name
        assert wait_for(self.is_running, timeout_seconds=30, interval_seconds=2), (
            "timed out waiting for %s container" % self.container_name
        )
        self.container.exec_run("start-minikube.sh", detach=True)
        self.connect_to_cluster(timeout)
        self.start_registry()

    def start_registry(self):
        self.get_client()
        print("\nStarting registry container localhost:%d in minikube ..." % self.registry_port)
        retry(
            p(
                self.client.containers.run,
                image="registry:2.7",
                name="registry",
                detach=True,
                environment={"REGISTRY_HTTP_ADDR": "0.0.0.0:%d" % self.registry_port},
                ports={"%d/tcp" % self.registry_port: self.registry_port},
            ),
            docker.errors.DockerException,
        )
        assert wait_for(
            p(tcp_socket_open, self.container_ip, self.registry_port), timeout_seconds=30, interval_seconds=2
        ), "timed out waiting for registry to start!"

    def pull_agent_image(self, name, tag, image_id=None):
        if image_id and has_docker_image(self.client, image_id):
            return self.client.images.get(image_id)

        if has_docker_image(self.client, name, tag):
            return self.client.images.get(name + ":" + tag)

        return self.client.images.pull(name, tag=tag)

    def build_image(self, dockerfile_dir, build_opts=None, docker_url=None):
        """
        Use low-level api client to build images in order to get build logs.
        Returns the image id.
        """

        def _build():
            client = docker.APIClient(base_url=docker_url, version="auto")
            build_log = []
            has_error = False
            image_id = None
            for line in client.build(path=dockerfile_dir, rm=True, forcerm=True, **build_opts):
                json_line = json.loads(line)
                keys = json_line.keys()
                if "stream" in keys:
                    build_log.append(json_line.get("stream").strip())
                else:
                    build_log.append(str(json_line))
                    if "error" in keys:
                        has_error = True
                    elif "aux" in keys:
                        image_id = json_line.get("aux").get("ID")
            assert not has_error, "build failed for %s:\n%s" % (dockerfile_dir, "\n".join(build_log))
            assert image_id, "failed to get id from output for built image:\n%s" % "\n".join(build_log)
            return image_id

        if os.path.isdir(os.path.join(TEST_SERVICES_DIR, dockerfile_dir)):
            dockerfile_dir = os.path.join(TEST_SERVICES_DIR, dockerfile_dir)
        else:
            assert os.path.isdir(dockerfile_dir), "Dockerfile directory %s not found!" % dockerfile_dir
        if build_opts is None:
            build_opts = {}
        if not docker_url:
            docker_url = "tcp://%s:2375" % self.container_ip
        print("\nBuilding image from %s ..." % dockerfile_dir)
        return retry(_build, AssertionError)

    def delete_resources(self):
        for doc in self.resources:
            kind = doc["kind"]
            name = doc["metadata"]["name"]
            namespace = doc["metadata"]["namespace"]
            api_client = k8s.api_client_from_version(doc["apiVersion"])
            if k8s.has_resource(name, kind, api_client, namespace):
                print('Deleting %s "%s" ...' % (kind, name))
                k8s.delete_resource(name, kind, api_client, namespace=namespace)

    @contextmanager
    def create_resources(self, yamls=None, namespace="default", timeout=k8s.K8S_CREATE_TIMEOUT):
        def wait_for_deployments():
            for doc in filter(lambda d: d["kind"] == "Deployment", self.resources):
                name = doc["metadata"]["name"]
                nspace = doc["metadata"]["namespace"]
                print("Waiting for deployment %s to be ready ..." % name)
                try:
                    start_time = time.time()
                    assert wait_for(
                        p(k8s.deployment_is_ready, name, nspace), timeout_seconds=timeout, interval_seconds=2
                    ), 'timed out waiting for deployment "%s" to be ready!\n%s' % (name, k8s.get_pod_logs(name, nspace))
                    print("Waited %d seconds" % (time.time() - start_time))
                finally:
                    print(self.exec_kubectl("describe deployment %s" % name, nspace))
                    for pod in k8s.get_all_pods(nspace):
                        print(self.exec_kubectl("describe pod %s" % pod.metadata.name, nspace))

        if yamls is None:
            yamls = []
        for yaml_file in yamls:
            assert os.path.isfile(yaml_file), '"%s" not found!' % yaml_file
            with open(yaml_file, "r") as fd:
                for doc in yaml.load_all(fd.read()):
                    kind = doc["kind"]
                    name = doc["metadata"]["name"]
                    nspace = doc["metadata"].setdefault("namespace", namespace)
                    api_client = k8s.api_client_from_version(doc["apiVersion"])
                    if k8s.has_resource(name, kind, api_client, namespace=nspace):
                        print('Deleting %s "%s" ...' % (kind, name))
                        k8s.delete_resource(name, kind, api_client, namespace=nspace)
                    print("Creating %s from %s ..." % (kind, yaml_file))
                    k8s.create_resource(doc, api_client, namespace=nspace, timeout=timeout)
                    self.resources.append(doc)

        wait_for_deployments()

        try:
            yield
        finally:
            self.delete_resources()
            self.resources = []

    @contextmanager
    def run_agent(self, agent_image, config=None, observer=None, monitors=None, namespace="default"):
        """
        Start the fake backend services and configure/create the k8s agent resources within the minikube container.

        Required Argument:
        agent_image:    Object returned from the agent_image fixture containing the agent image's name, tag, and id.

        Optional Arguments:
        config:         Configuration YAML for the agent (overwrites the configmap agent.yaml).
                        If not None, takes precedence over `observer` and `monitors` arguments (default: None).
        observer:       Name of the observer to set in the configmap agent.yaml (default: None).
        monitors:       List of monitors to set in the configmap agent.yaml (default: []).
        namespace:      Namespace for the agent (default: "default").
        """

        if not monitors:
            monitors = []
        with fake_backend.start(ip_addr=get_host_ip()) as backend:
            options = dict(
                image_name=agent_image["name"],
                image_tag=agent_image["tag"],
                observer=observer,
                monitors=monitors,
                config=config,
                cluster_name=self.cluster_name,
                namespace=namespace,
                backend=backend,
            )
            with self.agent.deploy(**options):
                try:
                    yield self.agent, backend
                finally:
                    if backend.datapoints:
                        print("\nDatapoints received:")
                        for dp in backend.datapoints:
                            print_dp_or_event(dp)
                    if backend.events:
                        print("\nEvents received:")
                        for event in backend.events:
                            print_dp_or_event(event)

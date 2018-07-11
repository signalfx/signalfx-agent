from contextlib import contextmanager
from functools import partial as p
from kubernetes import config as kube_config
from tests.helpers.util import *
from tests.kubernetes.agent import Agent
from tests.kubernetes.utils import *
import docker
import os
import tempfile
import time
import yaml

MINIKUBE_VERSION = os.environ.get("MINIKUBE_VERSION", "v0.26.1")
K8S_SERVICES_DIR = os.path.join(os.path.dirname(os.path.realpath(__file__)), 'services')
TEST_SERVICES_DIR = os.path.join(os.path.dirname(os.path.realpath(__file__)), "../../test-services")

class Minikube:
    def __init__(self):
        self.container = None
        self.client = None
        self.version = None
        self.name = None
        self.host_client = get_docker_client()
        self.yamls = []
        self.agent = Agent()
        self.cluster_name = "minikube"
        self.kubeconfig = None
        self.namespace = "default"
        self.worker_id = "master"

    def get_client(self):
        if self.container:
            self.container.reload()
            self.client = docker.DockerClient(base_url="tcp://%s:2375" % container_ip(self.container), version='auto')
            return self.client
        else:
            return None

    def load_kubeconfig(self, kubeconfig_path="/kubeconfig", timeout=300):
        with tempfile.NamedTemporaryFile(dir="/tmp/scratch") as fd:
            kubeconfig = fd.name
            assert wait_for(p(container_cmd_exit_0, self.container, "test -f %s" % kubeconfig_path), timeout_seconds=timeout), \
                "timed out waiting for the minikube cluster to be ready!\n\nMINIKUBE CONTAINER LOGS:\n%s\n\nLOCALKUBE LOGS:\n%s\n\n" % \
                (self.get_container_logs(), self.get_localkube_logs())
            time.sleep(2)
            rc, output = self.container.exec_run("cp -f %s %s" % (kubeconfig_path, kubeconfig))
            assert rc == 0, "failed to get %s from minikube!\n%s" % (kubeconfig_path, output.decode('utf-8'))
            self.kubeconfig = kubeconfig
            time.sleep(2)
            kube_config.load_kube_config(config_file=self.kubeconfig)

    def connect(self, name, timeout, version=None):
        print("\nConnecting to %s container ..." % name)
        assert wait_for(p(container_is_running, self.host_client, name), timeout_seconds=timeout), "timed out waiting for container %s!" % name
        self.container = self.host_client.containers.get(name)
        self.load_kubeconfig(timeout=timeout)
        self.client = self.get_client()
        self.name = name
        self.version = version

    def deploy(self, version, timeout, options={}):
        if container_is_running(self.host_client, "minikube"):
            self.host_client.containers.get("minikube").remove(force=True, v=True)
        self.version = version
        if self.version[0] != 'v':
            self.version = 'v' + self.version
        if not options:
            options = {
                "name": "minikube",
                "privileged": True,
                "extra_hosts": {
                    "localhost": get_host_ip()
                },
                "environment": {
                    'K8S_VERSION': self.version,
                    'TIMEOUT': str(timeout)
                },
                "ports": {
                    '8080/tcp': None,
                    '8443/tcp': None,
                    '2375/tcp': None,
                },
                "volumes": {
                    "/tmp/scratch": {
                        "bind": "/tmp/scratch",
                        "mode": "rw"
                    },
                }
            }
        print("\nDeploying minikube %s cluster ..." % self.version)
        image, logs = self.host_client.images.build(
            path=os.path.join(TEST_SERVICES_DIR, 'minikube'),
            buildargs={"MINIKUBE_VERSION": MINIKUBE_VERSION},
            tag="minikube:%s" % MINIKUBE_VERSION,
            rm=True,
            forcerm=True)
        self.container = self.host_client.containers.run(
            image.id,
            detach=True,
            **options)
        self.name = self.container.name
        self.load_kubeconfig(timeout=timeout)
        self.get_client()

    def build_image(self, dockerfile_dir, build_opts={}):
        if not self.client:
            self.get_client()
        self.client.images.build(
            path=dockerfile_dir,
            rm=True,
            forcerm=True,
            **build_opts)

    @contextmanager
    def deploy_k8s_yamls(self, yamls=[], namespace="default", timeout=180):
        self.yamls = []
        for yaml_file in yamls:
            assert os.path.isfile(yaml_file), "\"%s\" not found!" % yaml_file
            docs = []
            for doc in yaml.load_all(open(yaml_file, "r").read()):
                assert doc['kind'] in ["ConfigMap", "Deployment"], "kind \"%s\" in %s not yet supported!" % (doc['kind'], yaml_file)
                docs.append(doc)
            # create ConfigMaps first
            for doc in docs:
                kind = doc['kind']
                name = doc['metadata']['name']
                doc['metadata']['namespace'] = namespace
                if kind == "ConfigMap":
                    if has_configmap(name, namespace=namespace):
                        print("Deleting configmap \"%s\" ..." % name)
                        delete_configmap(name, namespace=namespace)
                    print("Creating configmap from %s ..." % yaml_file)
                    create_configmap(body=doc, namespace=namespace, timeout=timeout)
                    self.yamls.append(doc)
            # create Deployments
            for doc in docs:
                kind = doc['kind']
                name = doc['metadata']['name']
                doc['metadata']['namespace'] = namespace
                try:
                    containers = doc['spec']['template']['spec']['containers']
                    ports = []
                    for cont in containers:
                        for port in cont['ports']:
                            ports.append(int(port['containerPort']))
                except KeyError:
                    ports = []
                if kind == "ConfigMap":
                    continue
                if has_deployment(name, namespace=namespace):
                    print("Deleting deployment \"%s\" ..." % name)
                    delete_deployment(name, namespace=namespace)
                print("Creating deployment from %s ..." % yaml_file)
                create_deployment(body=doc, namespace=namespace, timeout=timeout)
                for port in ports:
                    for pod in get_all_pods_with_name(name, namespace=namespace):
                        assert wait_for(p(pod_port_open, self.container, pod.status.pod_ip, port), timeout_seconds=timeout), \
                            "timed out waiting for port %d for pod %s to be ready!" % (port, pod.metadata.name)
                self.yamls.append(doc)
        try:
            yield
        finally:
            for y in self.yamls:
                kind = y['kind']
                name = y['metadata']['name']
                namespace = y['metadata']['namespace']
                if kind == "ConfigMap":
                    print("Deleting configmap \"%s\" ..." % name)
                    delete_configmap(name, namespace=namespace)
                elif kind == "Deployment":
                    print("Deleting deployment \"%s\" ..." % name)
                    delete_deployment(name, namespace=namespace)
            self.yamls = []

    def pull_agent_image(self, name, tag):
        assert has_docker_image(self.host_client, name, tag), "agent image \"%s:%s\" not found!" % (name, tag)
        image_id = self.host_client.images.get("%s:%s" % (name, tag)).id
        if has_docker_image(self.client, image_id):
            return
        print("\nPulling %s:%s to the minikube container ..." % (name, tag))
        self.client.images.pull(name, tag=tag)
        _, output = self.container.exec_run('docker images')
        print(output.decode('utf-8'))

    @contextmanager
    def deploy_agent(self, configmap_path, daemonset_path, serviceaccount_path, observer=None, monitors=[], cluster_name="minikube", backend=None, image_name=None, image_tag=None, namespace="default"):
        self.pull_agent_image(image_name, image_tag)
        try:
            self.agent.deploy(self.client, configmap_path, daemonset_path, serviceaccount_path, observer, monitors, cluster_name=cluster_name, backend=backend, image_name=image_name, image_tag=image_tag, namespace=namespace)
        except Exception as e:
            print("\n\n%s\n\n" % get_all_logs(self))
            raise
        try:
            yield self.agent
        finally:
            print(self.agent.get_status())
            self.agent.delete()
            self.agent = Agent()

    def get_container_logs(self):
        try:
            return self.container.logs().decode('utf-8').strip()
        except Exception as e:
            return "Failed to get minikube container logs!\n%s" % str(e)

    def get_localkube_logs(self):
        try:
            rc, _ = self.container.exec_run("test -f /var/lib/localkube/localkube.err")
            if rc == 0:
                _, output = self.container.exec_run("cat /var/lib/localkube/localkube.err")
                return output.decode('utf-8').strip()
        except Exception as e:
            return "Failed to get localkube logs from minikube!\n%s" % str(e)

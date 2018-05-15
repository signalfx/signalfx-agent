from contextlib import contextmanager
from functools import partial as p
from kubernetes import config as kube_config
from tests.helpers.util import *
from tests.kubernetes.agent import Agent
from tests.kubernetes.utils import *
import docker
import os
import time
import yaml

MINIKUBE_VERSION = os.environ.get("MINIKUBE_VERSION", "v0.26.1")
K8S_SERVICES_DIR = os.path.join(os.path.dirname(os.path.realpath(__file__)), 'services')

class Minikube:
    def __init__(self):
        self.container = None
        self.client = None
        self.version = None
        self.name = None
        self.host_client = docker.from_env(version="auto")
        self.yamls = []
        self.agent = Agent()
        self.cluster_name = "minikube"
        self.kubeconfig = None
        self.namespace = "default"
        self.ip = None

    def get_client(self):
        if self.container:
            self.client = docker.DockerClient(base_url="tcp://%s:2375" % self.container.attrs["NetworkSettings"]["IPAddress"], version='auto')
            return self.client
        else:
            return None

    def get_ip(self):
        if self.container:
            self.ip = self.container.attrs["NetworkSettings"]["IPAddress"]
            return self.ip
        else:
            return None

    def load_kubeconfig(self, kubeconfig_path="/kubeconfig", timeout=300):
        assert wait_for(p(container_cmd_exit_0, self.container, "test -f %s" % kubeconfig_path), timeout_seconds=timeout), "timed out waiting for the minikube cluster to be ready!\n\n%s\n\n" % self.container.logs().decode('utf-8').strip()
        self.kubeconfig = "/tmp/scratch/kubeconfig-%s" % self.container.id[:12]
        time.sleep(2)
        rc, output = self.container.exec_run("cp -f %s %s" % (kubeconfig_path, self.kubeconfig))
        assert rc == 0, "failed to get %s from minikube!\n%s" % (kubeconfig_path, output.decode('utf-8'))
        time.sleep(2)
        kube_config.load_kube_config(config_file=self.kubeconfig)

    def connect(self, name, timeout, version=None):
        print("\nConnecting to %s container ..." % name)
        self.container = self.host_client.containers.get(name)
        self.client = self.get_client()
        self.name = name
        self.version = version
        self.load_kubeconfig(timeout=timeout)

    def deploy(self, version, timeout, name=None, options={}):
        self.version = version
        if self.version[0] != 'v':
            self.version = 'v' + self.version
        self.name = name
        if not self.name:
            self.name = "minikube-%s-%s" % (MINIKUBE_VERSION, self.version)
        if not options:
            options = {
                "name": self.name,
                "privileged": True,
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
        self.load_kubeconfig(timeout=timeout)
        self.container.reload()
        self.get_client()
        self.get_ip()

    def create_secret(self, key, secret):
        if self.container:
            rc, output = self.container.exec_run("kubectl create secret generic %s --from-literal=access-token=%s" % (key, secret))
            assert rc == 0
            print_lines(output.decode('utf-8'))

    @contextmanager
    def deploy_yamls(self, yamls=[], services_dir=K8S_SERVICES_DIR):
        self.yamls= []
        if services_dir:
            if len(yamls) == 0:
                yamls = sorted([os.path.join(services_dir, y) for y in os.listdir(services_dir) if y.endswith(".yaml")])
            else:
                yamls = sorted([os.path.join(services_dir, y) for y in yamls if y.endswith(".yaml")])
        for y in yamls:
            body = yaml.load(open(y))
            kind = body["kind"]
            name = body['metadata']['name']
            namespace = body['metadata']['namespace']
            if "configmap" in y:
                if has_configmap(name, namespace=namespace):
                    print("Deleting configmap \"%s\" ..." % name)
                    delete_configmap(name, namespace=namespace)
                print("Creating configmap from %s ..." % y)
                create_configmap(body=yaml.load(open(y)))
                self.yamls.append(body)
        for y in yamls:
            body = yaml.load(open(y))
            kind = body["kind"]
            name = body['metadata']['name']
            namespace = body['metadata']['namespace']
            body = yaml.load(open(y))
            if kind == "ConfigMap":
                continue
            assert kind == "Deployment", "kind \"%s\" in %s not yet supported!" % (kind, y)
            if has_deployment(name, namespace=namespace):
                print("Deleting deployment \"%s\" ..." % name)
                delete_deployment(name, namespace=namespace)
            print("Creating deployment from %s ..." % y)
            create_deployment(body=body)
            self.yamls.append(body)
        if len(self.yamls) > 0:
            assert wait_for(all_pods_have_ips, timeout_seconds=300), "timed out waiting for pod IPs!"
        try:
            yield
        finally:
            for y in self.yamls:
                kind = y["kind"]
                name = y['metadata']['name']
                namespace = y['metadata']['namespace']
                if kind == "ConfigMap":
                    print("Deleting configmap \"%s\" ..." % name)
                    delete_configmap(name, namespace=namespace)
                elif kind == "Deployment":
                    print("Deleting deployment \"%s\" ..." % name)
                    delete_deployment(name, namespace=namespace)
            self.yamls = []


    def pull_agent_image(self, name, tag="latest"):
        host_client = docker.from_env(version="auto")
        try:
            image_id = host_client.images.get("%s:%s" % (name, tag)).id
        except:
            image_id = None
        if image_id:
            try:
                self.client.images.get(image_id)
                return
            except:
                pass
        print("\nPulling %s:%s to the minikube container ..." % (name, tag))
        self.container.exec_run("cp -f /etc/hosts /etc/hosts.orig")
        self.container.exec_run("cp -f /etc/hosts /etc/hosts.new")
        self.container.exec_run("sed -i 's|127.0.0.1|%s|' /etc/hosts.new" % get_host_ip())
        self.container.exec_run("cp -f /etc/hosts.new /etc/hosts")
        time.sleep(5)
        self.client.images.pull(name, tag=tag)
        self.container.exec_run("cp -f /etc/hosts.orig /etc/hosts")
        _, output = self.container.exec_run('docker images')
        print_lines(output.decode('utf-8'))

    @contextmanager
    def deploy_agent(self, configmap_path, daemonset_path, serviceaccount_path, observer=None, monitors=[], cluster_name="minikube", backend=None, image_name=None, image_tag=None, namespace="default"):
        self.pull_agent_image(image_name, tag=image_tag)
        try:
            self.agent.deploy(self.client, configmap_path, daemonset_path, serviceaccount_path, observer, monitors, cluster_name=cluster_name, backend=backend, image_name=image_name, image_tag=image_tag, namespace=namespace)
        except Exception as e:
            print(str(e) + "\n\n%s\n\n" % get_all_logs(self))
            raise
        try:
            yield self.agent
        finally:
            self.agent.delete()
            self.agent = Agent()

    def get_container_logs(self):
        try:
            return self.container.logs().decode('utf-8').strip()
        except Exception as e:
            return "Failed to get minikube container logs!\n%s" % str(e)

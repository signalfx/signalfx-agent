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

MINIKUBE_VERSION = os.environ.get("MINIKUBE_VERSION", "v0.28.0")
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
        self.registry_port = None

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
        self.registry_port = get_free_port()
        if container_is_running(self.host_client, "minikube"):
            self.host_client.containers.get("minikube").remove(force=True, v=True)
        self.version = version
        if self.version[0] != 'v':
            self.version = 'v' + self.version
        if not options:
            options = {
                "name": "minikube",
                "privileged": True,
                "environment": {
                    'K8S_VERSION': self.version,
                    'TIMEOUT': str(timeout)
                },
                "ports": {
                    '8080/tcp': None,
                    '8443/tcp': None,
                    '2375/tcp': None,
                    '%d/tcp' % self.registry_port: self.registry_port,
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
        self.client = self.get_client()

    def start_registry(self):
        if not self.client:
            self.client = self.get_client()
        print("\nStarting registry container localhost:%d in minikube ..." % self.registry_port)
        self.client.containers.run(
            image='registry:latest',
            name="registry",
            detach=True,
            environment={"REGISTRY_HTTP_ADDR": "0.0.0.0:%d" % self.registry_port},
            ports={"%d/tcp" % self.registry_port: self.registry_port})

    def build_image(self, dockerfile_dir, build_opts={}):
        if not self.client:
            self.get_client()
        self.client.images.build(
            path=dockerfile_dir,
            rm=True,
            forcerm=True,
            **build_opts)

    @contextmanager
    def deploy_k8s_yamls(self, yamls=[], namespace=None, timeout=180):
        self.yamls = []
        for yaml_file in yamls:
            assert os.path.isfile(yaml_file), "\"%s\" not found!" % yaml_file
            docs = []
            with open(yaml_file, "r") as yf:
                docs = yaml.load_all(yf.read())

            for doc in docs:
                kind = doc['kind']
                name = doc['metadata']['name']
                api_version = doc['apiVersion']
                api_client = api_client_from_version(api_version)

                if not doc.get('metadata', {}).get('namespace'):
                    if 'metadata' not in doc:
                        doc['metadata'] = {}
                    doc['metadata']['namespace'] = namespace

                if has_resource(name, kind, api_client, namespace):
                    print("Deleting %s \"%s\" ..." % (kind, name))
                    delete_resource(name, kind, api_client, namespace=namespace)

                print("Creating %s from %s ..." % (kind, yaml_file))
                create_resource(doc, api_client, namespace=namespace, timeout=timeout)
                self.yamls.append(doc)

        for doc in filter(lambda d: d['kind'] == 'Deployment', self.yamls):
            print("Waiting for ports to open on deployment %s" % doc['metadata']['name'])
            wait_for_deployment(doc, self.container, timeout)

        try:
            yield
        finally:
            for y in self.yamls:
                print("Deleting %s \"%s\" ..." % (kind, name))

                kind = y['kind']
                api_version = y['apiVersion']
                api_client = api_client_from_version(api_version)
                delete_resource(name, kind, api_client, namespace=namespace)

            self.yamls = []

    def pull_agent_image(self, name, tag, image_id=None):
        if image_id and has_docker_image(self.client, image_id):
            return self.client.images.get(image_id)
        elif has_docker_image(self.client, name, tag):
            return self.client.images.get(name + ":" + tag)
        else:
            return self.client.images.pull(name, tag=tag)

    @contextmanager
    def deploy_agent(self, configmap_path, daemonset_path, serviceaccount_path, observer=None, monitors=[], cluster_name="minikube", backend=None, image_name=None, image_tag=None, namespace="default"):
        self.agent.deploy(
            self.client,
            configmap_path,
            daemonset_path,
            serviceaccount_path,
            observer,
            monitors,
            cluster_name=cluster_name,
            backend=backend,
            image_name=image_name,
            image_tag=image_tag,
            namespace=namespace)
        try:
            yield self.agent
            print("\n\n%s\n\n" % self.agent.get_status())
            print("\n\n%s\n\n" % self.agent.get_container_logs())
        except:
            print("\n\n%s\n\n" % get_all_logs(self))
            raise
        finally:
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

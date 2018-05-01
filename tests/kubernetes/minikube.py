from functools import partial as p
from kubernetes import config as kube_config
from tests.helpers.util import *
from tests.kubernetes.utils import *
import docker
import os
import time
import yaml

MINIKUBE_VERSION = os.environ.get("MINIKUBE_VERSION", "v0.26.1")
K8S_SERVICES_DIR = os.path.join(os.path.dirname(os.path.realpath(__file__)), 'services')

class Agent:
    def __init__(self):
        self.container = None
        self.serviceaccount_yaml = None
        self.configmap_yaml = None
        self.agent_yaml = None
        self.daemonset_yaml = None
        self.image_name = None
        self.image_tag = None
        self.backend = None
        self.observer = None
        self.monitors = []
        self.namespace = None
    
    def get_container(self, client, timeout=30):
        if not self.image_name or not self.image_tag:
            self.container = None
            return None
        start_time = time.time()
        while True:
            if (time.time() - start_time) > timeout:
                self.container = None
                return None
            try:
                self.container = client.containers.list(filters={"ancestor": self.image_name + ":" + self.image_tag})[0]
                return self.container
            except:
                time.sleep(2)

    def deploy(self, client, configmap_path, daemonset_path, serviceaccount_path, observer, monitors, cluster_name="minikube", backend=None, image_name=None, image_tag=None, namespace="default"):
        self.observer = observer
        self.monitors = monitors
        self.cluster_name = cluster_name
        self.backend = backend
        self.image_name = image_name
        self.image_tag = image_tag
        self.namespace = namespace
        print("\nDeploying signalfx-agent to the %s cluster ..." % cluster_name)
        if serviceaccount_path:
            self.serviceaccount_yaml = yaml.load(open(serviceaccount_path).read())
            create_serviceaccount(
                body=self.serviceaccount_yaml,
                namespace=namespace)
        self.configmap_yaml = yaml.load(open(configmap_path).read())
        self.agent_yaml = yaml.load(self.configmap_yaml['data']['agent.yaml'])
        del self.agent_yaml['observers']
        self.agent_yaml['observers'] = [{'type': observer}]
        self.agent_yaml['globalDimensions']['kubernetes_cluster'] = cluster_name
        self.agent_yaml['sendMachineID'] = True
        self.agent_yaml['useFullyQualifiedHost'] = False
        if backend:
            self.agent_yaml['ingestUrl'] = "http://%s:%d" % (get_host_ip(), backend.ingest_port)
            self.agent_yaml['apiUrl'] = "http://%s:%d" % (get_host_ip(), backend.api_port)
        if 'metricsToExclude' in self.agent_yaml.keys():
            del self.agent_yaml['metricsToExclude']
        del self.agent_yaml['monitors']
        self.agent_yaml['monitors'] = monitors
        self.configmap_yaml['data']['agent.yaml'] = yaml.dump(self.agent_yaml)
        create_configmap(
            body=self.configmap_yaml,
            namespace=namespace)
        self.daemonset_yaml = yaml.load(open(daemonset_path).read())
        if image_name and image_tag:
            self.daemonset_yaml['spec']['template']['spec']['containers'][0]['image'] = image_name + ":" + image_tag
        create_daemonset(
            body=self.daemonset_yaml,
            namespace=namespace)
        assert wait_for(p(has_pod, "signalfx-agent"), timeout_seconds=60), "timed out waiting for the signalfx-agent pod to start!"
        assert wait_for(all_pods_have_ips, timeout_seconds=300), "timed out waiting for pod IPs!"
        self.get_container(client)
        assert self.container, "failed to get agent container!"
        status = self.container.status.lower()
        # wait to make sure that the agent container is still running
        time.sleep(10)
        try:
            self.container.reload()
            status = self.container.status.lower()
        except:
            status = "exited"
        assert status == 'running', "agent container is not running!"
        return self

    def get_status(self):
        try:
            rc, output = self.container.exec_run("agent-status")
            if rc != 0:
                raise Exception(output.decode('utf-8').strip())
            return output.decode('utf-8').strip()
        except Exception as e:
            return "Failed to get agent-status!\n%s" % str(e)

    def get_container_logs(self):
        try:
            return self.container.logs().decode('utf-8').strip()
        except Exception as e:
            return "Failed to get agent container logs!\n%s" % str(e)

class Minikube:
    def __init__(self):
        self.container = None
        self.client = None
        self.version = None
        self.name = None
        self.host_client = docker.from_env(version="auto")
        self.services = []
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
            self.name = "minikube-%s" % self.version
        if not options:
            options = {
                "name": self.name,
                "privileged": True,
                "environment": {
                    'K8S_VERSION': self.version,
                    'TIMEOUT': str(timeout)
                },
                "ports": {
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
            buildargs={"MINIKUBE_VERSION": MINIKUBE_VERSION, "KUBECTL_VERSION": self.version},
            tag="minikube:%s-%s" % (MINIKUBE_VERSION, self.version),
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

    def deploy_services(self, services_dir=K8S_SERVICES_DIR):
        self.services = []
        yamls = [os.path.join(services_dir, y) for y in os.listdir(services_dir) if y.endswith(".yaml")]
        for y in yamls:
            if "configmap" in y:
                print("Creating configmap from %s ..." % y)
                create_configmap(body=yaml.load(open(y)))
        for y in yamls:
            body = yaml.load(open(y))
            if body["kind"] == "ConfigMap":
                continue
            assert body["kind"] in ["Deployment", "ReplicationController"], "kind \"%s\" in %s not yet supported!" % (body["kind"], y)
            if body["kind"] == "Deployment":
                print("Creating deployment from %s ..." % y)
                create_deployment(body=body)
            elif body["kind"] == "ReplicationController":
                print("Creating replication controller from %s ..." % y)
                create_replication_controller(body=body)
            self.services.append(body)
        if len(yamls) > 0:
            assert wait_for(all_pods_have_ips, timeout_seconds=300), "timed out waiting for pod IPs!"

    def pull_agent_image(self, name, tag=""):
        self.container.exec_run("cp -f /etc/hosts /etc/hosts.orig")
        self.container.exec_run("cp -f /etc/hosts /etc/hosts.new")
        self.container.exec_run("sed -i 's|127.0.0.1|%s|' /etc/hosts.new" % get_host_ip())
        self.container.exec_run("cp -f /etc/hosts.new /etc/hosts")
        time.sleep(5)
        self.client.images.pull(name, tag=tag)
        self.container.exec_run("cp -f /etc/hosts.orig /etc/hosts")
        _, output = self.container.exec_run('docker images')
        print_lines(output.decode('utf-8'))

    def deploy_agent(self, configmap_path, daemonset_path, serviceaccount_path, observer, monitors, cluster_name="minikube", backend=None, image_name=None, image_tag=None, namespace="default"):
        print("\nPulling %s:%s to the minikube container ..." % (image_name, image_tag))
        self.pull_agent_image(image_name, tag=image_tag)
        try:
            self.agent.deploy(self.client, configmap_path, daemonset_path, serviceaccount_path, observer, monitors, cluster_name=cluster_name, backend=backend, image_name=image_name, image_tag=image_tag, namespace=namespace)
        except Exception as e:
            print(str(e) + "\n\n%s\n\n" % get_all_logs(self))
            raise

    def get_container_logs(self):
        try:
            return self.container.logs().decode('utf-8').strip()
        except Exception as e:
            return "Failed to get minikube container logs!\n%s" % str(e)

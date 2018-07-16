from contextlib import contextmanager
from functools import partial as p
from tests.helpers.util import *
from tests.kubernetes.utils import *
import time
import yaml

class Agent:
    def __init__(self):
        self.container_name = None
        self.container = None
        self.serviceaccount_yaml = None
        self.configmap_yaml = None
        self.agent_yaml = None
        self.daemonset_yaml = None
        self.image_name = None
        self.image_tag = None
        self.backend = None
        self.observer = None
        self.cluster_name = None
        self.monitors = []
        self.namespace = None

    def get_container(self, client, timeout=30):
        assert wait_for(p(has_pod, self.container_name, namespace=self.namespace), timeout_seconds=timeout), "timed out waiting for \"%s\" pod!" % self.container_name
        pods = get_all_pods_starting_with_name(self.container_name, namespace=self.namespace)
        assert len(pods) == 1, "multiple pods found with name \"%s\"!\n%s" % (self.container_name, "\n".join([p.metadata.name for p in pods]))
        self.container = client.containers.get(pods[0].status.container_statuses[0].container_id.replace("docker:/", ""))

    def deploy(self, client, configmap_path, daemonset_path, serviceaccount_path, observer, monitors, cluster_name="minikube", backend=None, image_name=None, image_tag=None, namespace="default"):
        self.observer = observer
        self.monitors = monitors
        self.cluster_name = cluster_name
        self.backend = backend
        self.image_name = image_name
        self.image_tag = image_tag
        self.namespace = namespace
        self.serviceaccount_yaml = yaml.load(open(serviceaccount_path).read())
        self.serviceaccount_name = self.serviceaccount_yaml['metadata']['name']
        if not has_secret("signalfx-agent", namespace=self.namespace):
            print("Creating secret \"signalfx-agent\" ...")
            create_secret(
                "signalfx-agent",
                "access-token",
                "testing123",
                namespace=self.namespace)
        if not has_serviceaccount(self.serviceaccount_name, namespace=self.namespace):
            print("Creating service account \"%s\" from %s ..." % (self.serviceaccount_name, serviceaccount_path))
            create_serviceaccount(
                body=self.serviceaccount_yaml,
                namespace=self.namespace)
        self.configmap_yaml = yaml.load(open(configmap_path).read())
        self.configmap_name = self.configmap_yaml['metadata']['name']
        self.daemonset_yaml = yaml.load(open(daemonset_path).read())
        self.daemonset_name = self.daemonset_yaml['metadata']['name']
        self.container_name = self.daemonset_yaml['spec']['template']['spec']['containers'][0]['name']
        self.delete()
        self.agent_yaml = yaml.load(self.configmap_yaml['data']['agent.yaml'])
        del self.agent_yaml['observers']
        if not self.observer and "observers" in self.agent_yaml.keys():
            del self.agent_yaml['observers']
        elif self.observer == "k8s-api":
            self.agent_yaml['observers'] = [{'type': self.observer, "kubernetesAPI": {"authType": "serviceAccount", "skipVerify": False}}]
        elif self.observer == "k8s-kubelet":
            self.agent_yaml['observers'] = [{'type': self.observer, "kubeletAPI": {"authType": "serviceAccount", "skipVerify": True}}]
        elif self.observer == "docker":
            self.agent_yaml['observers'] = [{'type': self.observer, "dockerURL": "unix:///var/run/docker.sock"}]
        else:
            self.agent_yaml['observers'] = [{'type': self.observer}]
        self.agent_yaml['globalDimensions']['kubernetes_cluster'] = self.cluster_name
        self.agent_yaml['intervalSeconds'] = 5
        self.agent_yaml['sendMachineID'] = True
        self.agent_yaml['useFullyQualifiedHost'] = False
        if self.backend:
            self.agent_yaml['ingestUrl'] = "http://%s:%d" % (self.backend.ingest_host, self.backend.ingest_port)
            self.agent_yaml['apiUrl'] = "http://%s:%d" % (self.backend.api_host, self.backend.api_port)
        if 'metricsToExclude' in self.agent_yaml.keys():
            del self.agent_yaml['metricsToExclude']
        del self.agent_yaml['monitors']
        self.agent_yaml['monitors'] = self.monitors
        self.configmap_yaml['data']['agent.yaml'] = yaml.dump(self.agent_yaml)
        print("Creating configmap for observer=%s and monitor(s)=%s from %s ..." % (self.observer, ",".join([m["type"] for m in self.monitors]), configmap_path))
        create_configmap(
            body=self.configmap_yaml,
            namespace=self.namespace)
        if self.image_name and self.image_tag:
            print("Creating daemonset \"%s\" for %s:%s from %s ..." % (self.daemonset_name, self.image_name, self.image_tag, daemonset_path))
            self.daemonset_yaml['spec']['template']['spec']['containers'][0]['image'] = image_name + ":" + image_tag
        else:
            print("Creating daemonset \"%s\" from %s ..." % (self.daemonset_name, daemonset_path))
        create_daemonset(
            body=self.daemonset_yaml,
            namespace=self.namespace)
        assert wait_for(p(has_pod, self.daemonset_name, namespace=self.namespace), timeout_seconds=60), "timed out waiting for the %s pod to be created!" % self.daemonset_name
        #assert wait_for(p(all_pods_have_ips, namespace=self.namespace), timeout_seconds=300), "timed out waiting for pod IPs!"
        self.get_container(client)
        assert self.container, "failed to get agent container!"
        status = self.container.status.lower()
        # wait to make sure that the agent container is still running
        time.sleep(5)
        try:
            self.container.reload()
            status = self.container.status.lower()
        except:
            status = "exited"
        assert status == 'running', "agent container is not running!"
        return self

    def delete(self):
        if has_daemonset(self.daemonset_name, namespace=self.namespace):
            print("Deleting daemonset \"%s\" ..." % self.daemonset_name)
            delete_daemonset(self.daemonset_name, namespace=self.namespace)
            #assert wait_for(lambda: not has_pod(self.daemonset_name, namespace=self.namespace), timeout_seconds=60), \
            #    "timed out waiting for pod \"%s\" to be deleted!" % self.daemonset_name
        if has_configmap(self.configmap_name, namespace=self.namespace):
            print("Deleting configmap \"%s\" ..." % self.configmap_name)
            delete_configmap(self.configmap_name, namespace=self.namespace)
            #assert wait_for(lambda: not has_configmap(self.configmap_name, namespace=self.namespace), timeout_seconds=60), \
            #    "timed out waiting for configmap \"%s\" to be deleted!" % self.configmap_name
        
    def get_status(self):
        try:
            rc, output = self.container.exec_run("agent-status")
            if rc != 0:
                raise Exception(output.decode('utf-8').strip())
            return output.decode('utf-8').strip()
        except Exception as e:
            return "Failed to get agent-status!\n%s" % str(e)

    def get_container_logs(self):
        return get_pod_logs(
            self.container_name,
            namespace=self.namespace)

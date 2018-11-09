import os
from contextlib import contextmanager

import yaml
import tests.helpers.kubernetes.utils as k8s
from tests.helpers.util import ensure_always, get_unique_localhost

CUR_DIR = os.path.dirname(os.path.realpath(__file__))
AGENT_YAMLS_DIR = os.environ.get("AGENT_YAMLS_DIR", os.path.realpath(os.path.join(CUR_DIR, "../../../deployments/k8s")))
AGENT_CLUSTERROLE_PATH = os.environ.get("AGENT_CLUSTERROLE_PATH", os.path.join(AGENT_YAMLS_DIR, "clusterrole.yaml"))
AGENT_CLUSTERROLEBINDING_PATH = os.environ.get(
    "AGENT_CLUSTERROLEBINDING_PATH", os.path.join(AGENT_YAMLS_DIR, "clusterrolebinding.yaml")
)
AGENT_CONFIGMAP_PATH = os.environ.get("AGENT_CONFIGMAP_PATH", os.path.join(AGENT_YAMLS_DIR, "configmap.yaml"))
AGENT_DAEMONSET_PATH = os.environ.get("AGENT_DAEMONSET_PATH", os.path.join(AGENT_YAMLS_DIR, "daemonset.yaml"))
AGENT_SERVICEACCOUNT_PATH = os.environ.get(
    "AGENT_SERVICEACCOUNT_PATH", os.path.join(AGENT_YAMLS_DIR, "serviceaccount.yaml")
)
AGENT_STATUS_COMMAND = ["/bin/sh", "-c", "agent-status"]


class Agent:  # pylint: disable=too-many-instance-attributes
    def __init__(self):
        self.agent_yaml = None
        self.backend = None
        self.cluster_name = None
        self.clusterrole_name = None
        self.clusterrolebinding_name = None
        self.configmap_name = None
        self.daemonset_name = None
        self.image_name = None
        self.image_tag = None
        self.monitors = []
        self.observer = None
        self.namespace = None
        self.pods = []
        self.serviceaccount_name = None

    def create_agent_secret(self, secret="testing123"):
        if not k8s.has_secret("signalfx-agent", namespace=self.namespace):
            print('Creating secret "signalfx-agent" ...')
            k8s.create_secret("signalfx-agent", "access-token", secret, namespace=self.namespace)

    def create_agent_serviceaccount(self, serviceaccount_path):
        serviceaccount_yaml = yaml.load(open(serviceaccount_path).read())
        self.serviceaccount_name = serviceaccount_yaml["metadata"]["name"]
        if not k8s.has_serviceaccount(self.serviceaccount_name, namespace=self.namespace):
            print('Creating service account "%s" from %s ...' % (self.serviceaccount_name, serviceaccount_path))
            k8s.create_serviceaccount(body=serviceaccount_yaml, namespace=self.namespace)

    def create_agent_clusterrole(self, clusterrole_path, clusterrolebinding_path):
        clusterrole_yaml = yaml.load(open(clusterrole_path).read())
        self.clusterrole_name = clusterrole_yaml["metadata"]["name"]
        clusterrolebinding_yaml = yaml.load(open(clusterrolebinding_path).read())
        self.clusterrolebinding_name = clusterrolebinding_yaml["metadata"]["name"]
        if self.namespace != "default":
            self.clusterrole_name = self.clusterrole_name + "-" + self.namespace
            clusterrole_yaml["metadata"]["name"] = self.clusterrole_name
            self.clusterrolebinding_name = self.clusterrolebinding_name + "-" + self.namespace
            clusterrolebinding_yaml["metadata"]["name"] = self.clusterrolebinding_name
        if clusterrolebinding_yaml["roleRef"]["kind"] == "ClusterRole":
            clusterrolebinding_yaml["roleRef"]["name"] = self.clusterrole_name
        for subject in clusterrolebinding_yaml["subjects"]:
            subject["namespace"] = self.namespace
        if not k8s.has_clusterrole(self.clusterrole_name):
            print('Creating cluster role "%s" from %s ...' % (self.clusterrole_name, clusterrole_path))
            k8s.create_clusterrole(clusterrole_yaml)
        if not k8s.has_clusterrolebinding(self.clusterrolebinding_name):
            print(
                'Creating cluster role binding "%s" from %s ...'
                % (self.clusterrolebinding_name, clusterrolebinding_path)
            )
            k8s.create_clusterrolebinding(clusterrolebinding_yaml)

    def create_agent_configmap(self, configmap_path, agent_yaml=None):
        configmap_yaml = yaml.load(open(configmap_path).read())
        self.configmap_name = configmap_yaml["metadata"]["name"]
        self.delete_agent_configmap()
        if agent_yaml:
            self.agent_yaml = yaml.load(agent_yaml)
            self.observer = self.agent_yaml.setdefault("observers")
            self.monitors = self.agent_yaml.setdefault("monitors", [])
            self.agent_yaml.setdefault("globalDimensions", {"kubernetes_cluster": self.cluster_name})
            self.agent_yaml.setdefault("intervalSeconds", 5)
            self.agent_yaml.setdefault("sendMachineID", True)
            self.agent_yaml.setdefault("useFullyQualifiedHost", False)
            self.agent_yaml.setdefault("internalStatusHost", get_unique_localhost())
            if self.backend:
                self.agent_yaml.setdefault(
                    "ingestUrl", "http://%s:%d" % (self.backend.ingest_host, self.backend.ingest_port)
                )
                self.agent_yaml.setdefault("apiUrl", "http://%s:%d" % (self.backend.api_host, self.backend.api_port))
        else:
            self.agent_yaml = yaml.load(configmap_yaml["data"]["agent.yaml"])
            del self.agent_yaml["observers"]
            if not self.observer and self.agent_yaml.get("observers"):
                del self.agent_yaml["observers"]
            elif self.observer == "k8s-api":
                self.agent_yaml["observers"] = [
                    {"type": self.observer, "kubernetesAPI": {"authType": "serviceAccount", "skipVerify": False}}
                ]
            elif self.observer == "k8s-kubelet":
                self.agent_yaml["observers"] = [
                    {"type": self.observer, "kubeletAPI": {"authType": "serviceAccount", "skipVerify": True}}
                ]
            elif self.observer == "docker":
                self.agent_yaml["observers"] = [{"type": self.observer, "dockerURL": "unix:///var/run/docker.sock"}]
            else:
                self.agent_yaml["observers"] = [{"type": self.observer}]
            self.agent_yaml["globalDimensions"]["kubernetes_cluster"] = self.cluster_name
            self.agent_yaml["intervalSeconds"] = 5
            self.agent_yaml["sendMachineID"] = True
            self.agent_yaml["useFullyQualifiedHost"] = False
            self.agent_yaml["internalStatusHost"] = get_unique_localhost()
            if self.backend:
                self.agent_yaml["ingestUrl"] = "http://%s:%d" % (self.backend.ingest_host, self.backend.ingest_port)
                self.agent_yaml["apiUrl"] = "http://%s:%d" % (self.backend.api_host, self.backend.api_port)
            if self.agent_yaml.get("metricsToExclude"):
                del self.agent_yaml["metricsToExclude"]
            del self.agent_yaml["monitors"]
            self.agent_yaml["monitors"] = self.monitors
        configmap_yaml["data"]["agent.yaml"] = yaml.dump(self.agent_yaml)
        print(
            "Creating configmap for observer=%s and monitor(s)=%s from %s ..."
            % (self.observer, ",".join([m["type"] for m in self.monitors]), configmap_path)
        )
        k8s.create_configmap(body=configmap_yaml, namespace=self.namespace)
        print(self.agent_yaml)

    def create_agent_daemonset(self, daemonset_path):
        daemonset_yaml = yaml.load(open(daemonset_path).read())
        self.daemonset_name = daemonset_yaml["metadata"]["name"]
        daemonset_labels = daemonset_yaml["spec"]["selector"]["matchLabels"]
        self.delete_agent_daemonset()
        daemonset_yaml["spec"]["template"]["spec"]["containers"][0]["resources"] = {"requests": {"cpu": "100m"}}
        if self.image_name and self.image_tag:
            print(
                'Creating daemonset "%s" for %s:%s from %s ...'
                % (self.daemonset_name, self.image_name, self.image_tag, daemonset_path)
            )
            daemonset_yaml["spec"]["template"]["spec"]["containers"][0]["image"] = (
                self.image_name + ":" + self.image_tag
            )
        else:
            print('Creating daemonset "%s" from %s ...' % (self.daemonset_name, daemonset_path))
        k8s.create_daemonset(body=daemonset_yaml, namespace=self.namespace)
        assert ensure_always(lambda: k8s.daemonset_is_ready(self.daemonset_name, namespace=self.namespace), 5)
        labels = ",".join(["%s=%s" % keyval for keyval in daemonset_labels.items()])
        self.pods = k8s.get_pods_by_labels(labels, namespace=self.namespace)
        assert self.pods, "no agent pods found"
        assert all([k8s.pod_is_ready(pod.metadata.name, namespace=self.namespace) for pod in self.pods])

    @contextmanager
    def deploy(self, **kwargs):
        self.observer = kwargs.get("observer")
        self.monitors = kwargs.get("monitors")
        self.cluster_name = kwargs.get("cluster_name", "minikube")
        self.backend = kwargs.get("backend")
        self.image_name = kwargs.get("image_name")
        self.image_tag = kwargs.get("image_tag")
        self.namespace = kwargs.get("namespace", "default")

        self.create_agent_secret()
        self.create_agent_serviceaccount(kwargs.get("serviceaccount_path", AGENT_SERVICEACCOUNT_PATH))
        self.create_agent_clusterrole(
            kwargs.get("clusterrole_path", AGENT_CLUSTERROLE_PATH),
            kwargs.get("clusterrolebinding_path", AGENT_CLUSTERROLEBINDING_PATH),
        )
        self.create_agent_configmap(kwargs.get("configmap_path", AGENT_CONFIGMAP_PATH), kwargs.get("config"))
        self.create_agent_daemonset(kwargs.get("daemonset_path", AGENT_DAEMONSET_PATH))

        try:
            yield self
        finally:
            print("\nAgent status:\n%s" % self.get_status())
            print("\nAgent logs:\n%s" % self.get_logs())
            self.delete()
            self.__init__()

    def delete_agent_daemonset(self):
        if self.daemonset_name and k8s.has_daemonset(self.daemonset_name, namespace=self.namespace):
            print('Deleting daemonset "%s" ...' % self.daemonset_name)
            k8s.delete_daemonset(self.daemonset_name, namespace=self.namespace)

    def delete_agent_configmap(self):
        if self.configmap_name and k8s.has_configmap(self.configmap_name, namespace=self.namespace):
            print('Deleting configmap "%s" ...' % self.configmap_name)
            k8s.delete_configmap(self.configmap_name, namespace=self.namespace)

    def delete(self):
        self.delete_agent_daemonset()
        self.delete_agent_configmap()

    def get_status(self, command=None):
        if not command:
            command = AGENT_STATUS_COMMAND
        output = ""
        for pod in self.pods:
            output += "pod/%s:\n" % pod.metadata.name
            output += k8s.exec_pod_command(pod.metadata.name, command, namespace=self.namespace) + "\n"
        return output.strip()

    def get_logs(self):
        output = ""
        for pod in self.pods:
            output += "pod/%s\n" % pod.metadata.name
            output += k8s.get_pod_logs(pod.metadata.name, namespace=self.namespace) + "\n"
        return output.strip()

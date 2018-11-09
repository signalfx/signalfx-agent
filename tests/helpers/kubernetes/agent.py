import os
import time
from contextlib import contextmanager
from functools import partial as p

import yaml
from tests.helpers.kubernetes.utils import (
    create_clusterrole,
    create_clusterrolebinding,
    create_configmap,
    create_daemonset,
    create_secret,
    create_serviceaccount,
    delete_configmap,
    delete_daemonset,
    get_all_pods_starting_with_name,
    get_pod_logs,
    has_clusterrole,
    has_clusterrolebinding,
    has_configmap,
    has_daemonset,
    has_pod,
    has_secret,
    has_serviceaccount,
)
from tests.helpers.util import get_internal_status_host, wait_for

CUR_DIR = os.path.dirname(os.path.realpath(__file__))
AGENT_YAMLS_DIR = os.environ.get("AGENT_YAMLS_DIR", os.path.realpath(os.path.join(CUR_DIR, "../../../deployments/k8s")))
AGENT_CONFIGMAP_PATH = os.environ.get("AGENT_CONFIGMAP_PATH", os.path.join(AGENT_YAMLS_DIR, "configmap.yaml"))
AGENT_DAEMONSET_PATH = os.environ.get("AGENT_DAEMONSET_PATH", os.path.join(AGENT_YAMLS_DIR, "daemonset.yaml"))
AGENT_SERVICEACCOUNT_PATH = os.environ.get(
    "AGENT_SERVICEACCOUNT_PATH", os.path.join(AGENT_YAMLS_DIR, "serviceaccount.yaml")
)
AGENT_CLUSTERROLE_PATH = os.environ.get("AGENT_CLUSTERROLE_PATH", os.path.join(AGENT_YAMLS_DIR, "clusterrole.yaml"))
AGENT_CLUSTERROLEBINDING_PATH = os.environ.get(
    "AGENT_CLUSTERROLEBINDING_PATH", os.path.join(AGENT_YAMLS_DIR, "clusterrolebinding.yaml")
)


class Agent:  # pylint: disable=too-many-instance-attributes
    def __init__(self):
        self.agent_yaml = None
        self.backend = None
        self.cluster_name = None
        self.clusterrole_name = None
        self.clusterrole_yaml = None
        self.clusterrolebinding_name = None
        self.clusterrolebinding_yaml = None
        self.configmap_name = None
        self.configmap_yaml = None
        self.container = None
        self.container_name = None
        self.daemonset_name = None
        self.daemonset_yaml = None
        self.image_name = None
        self.image_tag = None
        self.monitors = []
        self.observer = None
        self.namespace = None
        self.serviceaccount_name = None
        self.serviceaccount_yaml = None

    def get_container(self, client, timeout=30):
        assert wait_for(p(has_pod, self.container_name, namespace=self.namespace), timeout_seconds=timeout), (
            'timed out waiting for "%s" pod!' % self.container_name
        )
        pods = get_all_pods_starting_with_name(self.container_name, namespace=self.namespace)
        assert len(pods) == 1, 'multiple pods found with name "%s"!\n%s' % (
            self.container_name,
            "\n".join([p.metadata.name for p in pods]),
        )
        self.container = client.containers.get(
            pods[0].status.container_statuses[0].container_id.replace("docker:/", "")
        )

    def create_agent_secret(self, secret="testing123"):
        if not has_secret("signalfx-agent", namespace=self.namespace):
            print('Creating secret "signalfx-agent" ...')
            create_secret("signalfx-agent", "access-token", secret, namespace=self.namespace)

    def create_agent_serviceaccount(self, serviceaccount_path):
        self.serviceaccount_yaml = yaml.load(open(serviceaccount_path).read())
        self.serviceaccount_name = self.serviceaccount_yaml["metadata"]["name"]
        if not has_serviceaccount(self.serviceaccount_name, namespace=self.namespace):
            print('Creating service account "%s" from %s ...' % (self.serviceaccount_name, serviceaccount_path))
            create_serviceaccount(body=self.serviceaccount_yaml, namespace=self.namespace)

    def create_agent_clusterrole(self, clusterrole_path, clusterrolebinding_path):
        self.clusterrole_yaml = yaml.load(open(clusterrole_path).read())
        self.clusterrole_name = self.clusterrole_yaml["metadata"]["name"]
        self.clusterrolebinding_yaml = yaml.load(open(clusterrolebinding_path).read())
        self.clusterrolebinding_name = self.clusterrolebinding_yaml["metadata"]["name"]
        if self.namespace != "default":
            self.clusterrole_name = self.clusterrole_name + "-" + self.namespace
            self.clusterrole_yaml["metadata"]["name"] = self.clusterrole_name
            self.clusterrolebinding_name = self.clusterrolebinding_name + "-" + self.namespace
            self.clusterrolebinding_yaml["metadata"]["name"] = self.clusterrolebinding_name
        if self.clusterrolebinding_yaml["roleRef"]["kind"] == "ClusterRole":
            self.clusterrolebinding_yaml["roleRef"]["name"] = self.clusterrole_name
        for subject in self.clusterrolebinding_yaml["subjects"]:
            subject["namespace"] = self.namespace
        if not has_clusterrole(self.clusterrole_name):
            print('Creating cluster role "%s" from %s ...' % (self.clusterrole_name, clusterrole_path))
            create_clusterrole(self.clusterrole_yaml)
        if not has_clusterrolebinding(self.clusterrolebinding_name):
            print(
                'Creating cluster role binding "%s" from %s ...'
                % (self.clusterrolebinding_name, clusterrolebinding_path)
            )
            create_clusterrolebinding(self.clusterrolebinding_yaml)

    def create_agent_configmap(self, configmap_path):
        self.configmap_yaml = yaml.load(open(configmap_path).read())
        self.configmap_name = self.configmap_yaml["metadata"]["name"]
        self.delete_agent_configmap()
        self.agent_yaml = yaml.load(self.configmap_yaml["data"]["agent.yaml"])
        del self.agent_yaml["observers"]
        if not self.observer and "observers" in self.agent_yaml.keys():
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
        self.agent_yaml["internalStatusHost"] = get_internal_status_host()
        if self.backend:
            self.agent_yaml["ingestUrl"] = "http://%s:%d" % (self.backend.ingest_host, self.backend.ingest_port)
            self.agent_yaml["apiUrl"] = "http://%s:%d" % (self.backend.api_host, self.backend.api_port)
        if "metricsToExclude" in self.agent_yaml.keys():
            del self.agent_yaml["metricsToExclude"]
        del self.agent_yaml["monitors"]
        self.agent_yaml["monitors"] = self.monitors
        self.configmap_yaml["data"]["agent.yaml"] = yaml.dump(self.agent_yaml)
        print(
            "Creating configmap for observer=%s and monitor(s)=%s from %s ..."
            % (self.observer, ",".join([m["type"] for m in self.monitors]), configmap_path)
        )
        create_configmap(body=self.configmap_yaml, namespace=self.namespace)

    def create_agent_daemonset(self, daemonset_path):
        self.daemonset_yaml = yaml.load(open(daemonset_path).read())
        self.daemonset_name = self.daemonset_yaml["metadata"]["name"]
        self.delete_agent_daemonset()
        self.container_name = self.daemonset_yaml["spec"]["template"]["spec"]["containers"][0]["name"]
        self.daemonset_yaml["spec"]["template"]["spec"]["containers"][0]["resources"] = {"requests": {"cpu": "100m"}}
        if self.image_name and self.image_tag:
            print(
                'Creating daemonset "%s" for %s:%s from %s ...'
                % (self.daemonset_name, self.image_name, self.image_tag, daemonset_path)
            )
            self.daemonset_yaml["spec"]["template"]["spec"]["containers"][0]["image"] = (
                self.image_name + ":" + self.image_tag
            )
        else:
            print('Creating daemonset "%s" from %s ...' % (self.daemonset_name, daemonset_path))
        create_daemonset(body=self.daemonset_yaml, namespace=self.namespace)
        assert wait_for(p(has_pod, self.daemonset_name, namespace=self.namespace), timeout_seconds=60), (
            "timed out waiting for the %s pod to be created!" % self.daemonset_name
        )

    @contextmanager
    def deploy(
        self,
        client,
        configmap_path=AGENT_CONFIGMAP_PATH,
        daemonset_path=AGENT_DAEMONSET_PATH,
        serviceaccount_path=AGENT_SERVICEACCOUNT_PATH,
        clusterrole_path=AGENT_CLUSTERROLE_PATH,
        clusterrolebinding_path=AGENT_CLUSTERROLEBINDING_PATH,
        observer=None,
        monitors=None,
        cluster_name="minikube",
        backend=None,
        image_name=None,
        image_tag=None,
        namespace="default",
    ):  # pylint: disable=too-many-arguments,too-many-locals
        self.observer = observer
        self.monitors = monitors
        self.cluster_name = cluster_name
        self.backend = backend
        self.image_name = image_name
        self.image_tag = image_tag
        self.namespace = namespace

        self.create_agent_secret()
        self.create_agent_serviceaccount(serviceaccount_path)
        self.create_agent_clusterrole(clusterrole_path, clusterrolebinding_path)
        self.create_agent_configmap(configmap_path)
        self.create_agent_daemonset(daemonset_path)

        self.get_container(client)
        assert self.container, "failed to get agent container!"
        status = self.container.status.lower()
        # wait to make sure that the agent container is still running
        time.sleep(5)
        try:
            self.container.reload()
            status = self.container.status.lower()
        except Exception:  # pylint: disable=broad-except
            status = "exited"
        assert status == "running", "agent container is not running!"
        yield self

    def delete_agent_daemonset(self):
        if self.daemonset_name and has_daemonset(self.daemonset_name, namespace=self.namespace):
            print('Deleting daemonset "%s" ...' % self.daemonset_name)
            delete_daemonset(self.daemonset_name, namespace=self.namespace)

    def delete_agent_configmap(self):
        if self.configmap_name and has_configmap(self.configmap_name, namespace=self.namespace):
            print('Deleting configmap "%s" ...' % self.configmap_name)
            delete_configmap(self.configmap_name, namespace=self.namespace)

    def delete(self):
        self.delete_agent_daemonset()
        self.delete_agent_configmap()

    def get_status(self):
        try:
            code, output = self.container.exec_run("agent-status")
            if code != 0:
                raise Exception(output.decode("utf-8").strip())
            return output.decode("utf-8").strip()
        except Exception as e:  # pylint: disable=broad-except
            return "Failed to get agent-status!\n%s" % str(e)

    def get_container_logs(self):
        return get_pod_logs(self.container_name, namespace=self.namespace)

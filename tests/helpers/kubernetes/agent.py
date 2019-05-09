import os
from contextlib import contextmanager
from pathlib import Path

import yaml
from kubernetes import client as kube_client
from tests import paths
from tests.helpers.kubernetes import utils
from tests.helpers.util import get_unique_localhost

AGENT_YAMLS_DIR = Path(os.environ.get("AGENT_YAMLS_DIR", paths.REPO_ROOT_DIR / "deployments" / "k8s"))
AGENT_CLUSTERROLE_PATH = Path(os.environ.get("AGENT_CLUSTERROLE_PATH", AGENT_YAMLS_DIR / "clusterrole.yaml"))
AGENT_CLUSTERROLEBINDING_PATH = Path(
    os.environ.get("AGENT_CLUSTERROLEBINDING_PATH", AGENT_YAMLS_DIR / "clusterrolebinding.yaml")
)
AGENT_CONFIGMAP_PATH = Path(os.environ.get("AGENT_CONFIGMAP_PATH", AGENT_YAMLS_DIR / "configmap.yaml"))
AGENT_DAEMONSET_PATH = Path(os.environ.get("AGENT_DAEMONSET_PATH", AGENT_YAMLS_DIR / "daemonset.yaml"))
AGENT_SERVICEACCOUNT_PATH = Path(os.environ.get("AGENT_SERVICEACCOUNT_PATH", AGENT_YAMLS_DIR / "serviceaccount.yaml"))
AGENT_STATUS_COMMAND = ["/bin/sh", "-c", "agent-status all"]


def load_resource_yaml(path):
    with open(path, "r") as fd:
        return yaml.safe_load(fd.read())


class Agent:
    def __init__(self, agent_image_name, cluster, namespace, fake_services, fake_services_pod_ip):
        assert cluster, "Agent on K8s must be associated with a cluster"
        self.cluster = cluster

        self.namespace = namespace
        self.agent_image_name = agent_image_name

        self.fake_services = fake_services
        self.fake_services_pod_ip = fake_services_pod_ip

        self.pods = []

    def fill_in_configmap(self, configmap, agent_yaml):
        agent_conf = yaml.safe_load(agent_yaml)
        agent_conf.setdefault("observers", [])
        agent_conf.setdefault("monitors", [])
        agent_conf.setdefault("globalDimensions", {"kubernetes_cluster": self.namespace})
        agent_conf.setdefault("intervalSeconds", 5)
        agent_conf.setdefault("enableBuiltInFiltering", True)
        agent_conf.setdefault("ingestUrl", f"http://{self.fake_services_pod_ip}:{self.fake_services.ingest_port}")
        agent_conf.setdefault("apiUrl", f"http://{self.fake_services_pod_ip}:{self.fake_services.api_port}")
        agent_conf.setdefault("internalStatusHost", get_unique_localhost())

        configmap["data"]["agent.yaml"] = yaml.dump(agent_conf)

    @contextmanager
    def deploy_unique_rbac_resources(self):
        """
        The cluster-wide RBAC resources (clusterrole/clusterroldbinding) are
        not namespaced, so they have to be handled specially to ensure they are
        unique amongst potentially multiple deployments of the agent in the
        same cluster.  Basically just sticks the test namespace as a suffix to
        the resource names.
        """
        corev1 = kube_client.CoreV1Api()
        rbacv1beta1 = kube_client.RbacAuthorizationV1beta1Api()

        serviceaccount = corev1.create_namespaced_service_account(
            body=load_resource_yaml(AGENT_SERVICEACCOUNT_PATH), namespace=self.namespace
        )

        clusterrole_base = load_resource_yaml(AGENT_CLUSTERROLE_PATH)
        clusterrole_base["metadata"]["name"] = f"signalfx-agent-{self.namespace}"
        clusterrole = rbacv1beta1.create_cluster_role(body=clusterrole_base)

        crb_base = load_resource_yaml(AGENT_CLUSTERROLEBINDING_PATH)
        # Make the binding refer to our testing namespace's role and service account
        crb_base["metadata"]["name"] = f"signalfx-agent-{self.namespace}"
        crb_base["roleRef"]["name"] = clusterrole.metadata.name
        crb_base["subjects"][0]["namespace"] = self.namespace
        crb = rbacv1beta1.create_cluster_role_binding(body=crb_base)

        try:
            yield
        finally:
            delete_opts = kube_client.V1DeleteOptions(grace_period_seconds=0, propagation_policy="Background")

            rbacv1beta1.delete_cluster_role_binding(crb.metadata.name, body=delete_opts)
            rbacv1beta1.delete_cluster_role(clusterrole.metadata.name, body=delete_opts)
            corev1.delete_namespaced_service_account(
                serviceaccount.metadata.name, namespace=self.namespace, body=delete_opts
            )
            print("Deleted RBAC resources")

    @contextmanager
    def deploy(self, agent_yaml=None):
        with self.deploy_unique_rbac_resources():
            secret = None
            daemonset = None
            configmap = None
            try:
                secret = utils.create_secret("signalfx-agent", "access-token", "testing123", namespace=self.namespace)
                print("Created agent secret")

                configmap_base = load_resource_yaml(AGENT_CONFIGMAP_PATH)
                self.fill_in_configmap(configmap_base, agent_yaml)
                configmap = utils.create_configmap(body=configmap_base, namespace=self.namespace)
                print(f"Created agent configmap:\n{configmap_base}")

                daemonset_base = load_resource_yaml(AGENT_DAEMONSET_PATH)
                daemonset_base["spec"]["template"]["spec"]["containers"][0]["image"] = self.agent_image_name
                daemonset_base["spec"]["template"]["spec"]["containers"][0]["imagePullPolicy"] = "Always"
                daemonset_base["spec"]["template"]["spec"]["containers"][0]["resources"] = {"requests": {"cpu": "50m"}}
                daemonset = utils.create_daemonset(body=daemonset_base, namespace=self.namespace)
                print(f"Created agent daemonset:\n{daemonset_base}")

                yield
            finally:
                print("\nAgent status:\n%s" % self.get_status())
                print("\nAgent logs:\n%s" % self.get_logs())
                if daemonset:
                    utils.delete_daemonset(daemonset.metadata.name, namespace=self.namespace)
                if configmap:
                    utils.delete_configmap(configmap.metadata.name, namespace=self.namespace)
                if secret:
                    corev1 = kube_client.CoreV1Api()
                    corev1.delete_namespaced_secret(
                        name=secret.metadata.name,
                        body=kube_client.V1DeleteOptions(grace_period_seconds=0, propagation_policy="Background"),
                        namespace=secret.metadata.namespace,
                    )

    def get_agent_pods(self):
        return utils.get_pods_by_labels("app=signalfx-agent", namespace=self.namespace)

    def get_status(self, command=None):
        if not command:
            command = AGENT_STATUS_COMMAND
        output = ""
        for pod in self.get_agent_pods():
            output += "pod/%s:\n" % pod.metadata.name
            output += utils.exec_pod_command(pod.metadata.name, command, namespace=self.namespace) + "\n"
        return output.strip()

    def get_logs(self):
        output = ""
        for pod in self.get_agent_pods():
            output += "pod/%s\n" % pod.metadata.name
            output += utils.get_pod_logs(pod.metadata.name, namespace=self.namespace) + "\n"
        return output.strip()

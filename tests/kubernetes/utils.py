from functools import partial as p
from kubernetes import client as kube_client
from kubernetes.client.rest import ApiException
from tests.helpers.assertions import *
from tests.helpers.util import *
import docker
import netifaces as ni
import os
import re
import socket
import time
import yaml

CUR_DIR = os.path.dirname(os.path.realpath(__file__))
AGENT_YAMLS_DIR = os.environ.get("AGENT_YAMLS_DIR", os.path.join(CUR_DIR, "../../deployments/k8s"))
AGENT_CONFIGMAP_PATH = os.environ.get("AGENT_CONFIGMAP_PATH", os.path.join(AGENT_YAMLS_DIR, "configmap.yaml"))
AGENT_DAEMONSET_PATH = os.environ.get("AGENT_DAEMONSET_PATH", os.path.join(AGENT_YAMLS_DIR, "daemonset.yaml"))
AGENT_SERVICEACCOUNT_PATH = os.environ.get("AGENT_SERVICEACCOUNT_PATH", os.path.join(AGENT_YAMLS_DIR, "serviceaccount.yaml"))
K8S_CREATE_TIMEOUT = 180
K8S_DELETE_TIMEOUT = 10


def run_k8s_monitors_test(agent_image, minikube, monitors, observer=None, namespace="default", yamls=[], yamls_timeout=K8S_CREATE_TIMEOUT, expected_metrics=set(), expected_dims=set(), passwords=["testing123"], test_timeout=60):
    """
    Wrapper function for K8S setup and tests within minikube for monitors.
    Setup includes starting the fake backend, creating K8S deployments, and smart agent configuration/deployment within minikube.
    Tests include waiting for at least one metric and/or dimension from the expected_metrics and expected_dims args, and checking for
    cleartext passwords in the output from the agent status and agent container logs.

    Args:
    agent_image (dict):                    Dict object from the agent_image fixture
    minikube (Minikube):                   Minkube object from the minikube fixture
    monitors (str, dict, or list of dict): YAML-based definition of monitor(s) for the smart agent agent.yaml
    observer (str):                        Observer for the smart agent agent.yaml (if None, the agent.yaml will not be configured for an observer)
    namespace (str):                       K8S namespace for the smart agent and deployments
    yamls (list of str):                   Path(s) to K8S deployment yamls to create
    yamls_timeout (int):                   Timeout in seconds to wait for the K8S deployments to be ready
    expected_metrics (set of str):         Set of metric names to test for (if empty or None, metrics test will be skipped)
    expected_dims (set or str):            Set of dimension keys to test for (if None, dimensions test will be skipped)
    passwords (list of str):               Cleartext password(s) to test for in the output from the agent status and agent container logs
    test_timeout (int):                    Timeout in seconds to wait for metrics/dimensions
    """
    if expected_dims is not None:
        if observer:
            expected_dims = expected_dims.union(get_observer_dims_from_selfdescribe(observer))
        expected_dims = expected_dims.union({"kubernetes_cluster"})
    try:
        monitors = yaml.load(monitors)
    except AttributeError:
        pass
    if type(monitors) is dict:
        monitors = [monitors]
    assert type(monitors) is list, "unknown type/defintion for monitors:\n%s\n" % monitors
    with fake_backend.start(ip=get_host_ip()) as backend:
        with minikube.deploy_k8s_yamls(yamls, namespace=namespace, timeout=yamls_timeout):
            with minikube.deploy_agent(
                AGENT_CONFIGMAP_PATH,
                AGENT_DAEMONSET_PATH,
                AGENT_SERVICEACCOUNT_PATH,
                observer=observer,
                monitors=monitors,
                cluster_name="minikube",
                backend=backend,
                image_name=agent_image["name"],
                image_tag=agent_image["tag"],
                namespace=namespace) as agent:
                    result = has_any_metric_or_dim(backend, expected_metrics, expected_dims, timeout=test_timeout)
                    if expected_metrics and expected_dims:
                        assert result, \
                            "timed out waiting for any metric in %s with any dimension key in %s!\n\nAGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n" % \
                            (expected_metrics, expected_dims, agent.get_status(), agent.get_container_logs())
                    elif expected_metrics:
                        assert result, \
                            "timed out waiting for any metric in %s!\n\nAGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n" % \
                            (expected_metrics, agent.get_status(), agent.get_container_logs())
                    elif expected_dims:
                        assert result, \
                            "timed out waiting for any dimension key in %s!\n\nAGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n" % \
                            (expected_dims, agent.get_status(), agent.get_container_logs())
                    agent_status = agent.get_status()
                    container_logs = agent.get_container_logs()
                    assert all([p not in agent_status for p in passwords]), \
                        "cleartext password(s) found in agent-status output!\n\n%s\n" % agent_status
                    assert all([p not in container_logs for p in passwords]), \
                        "cleartext password(s) found in agent container logs!\n\n%s\n" % container_logs


def get_host_ip():
    gws = ni.gateways()
    interface = gws['default'][ni.AF_INET][1]
    return ni.ifaddresses(interface)[ni.AF_INET][0]['addr']


def has_namespace(name):
    api = kube_client.CoreV1Api()
    try:
        api.read_namespace(name=name)
        return True
    except ApiException as e:
        if e.status == 404:
            return False
        else:
            raise


def create_namespace(name):
    api = kube_client.CoreV1Api()
    return api.create_namespace(
        body=kube_client.V1Namespace(
            metadata=kube_client.V1ObjectMeta(name=name)))

    
def has_secret(name, namespace="default"):
    api = kube_client.CoreV1Api()
    try:
        api.read_namespaced_secret(name, namespace=namespace)
        return True
    except ApiException as e:
        if e.status == 404:
            return False
        else:
            raise


def create_secret(name, key, value, namespace="default"):
    if not has_namespace(namespace):
        create_namespace(namespace)
    api = kube_client.CoreV1Api()
    return api.create_namespaced_secret(
        body=kube_client.V1Secret(
            metadata=kube_client.V1ObjectMeta(name=name, namespace=namespace),
            string_data={key: value}),
        namespace=namespace)


def has_serviceaccount(name, namespace="default"):
    api = kube_client.CoreV1Api()
    try:
        api.read_namespaced_service_account(name, namespace=namespace)
        return True
    except ApiException as e:
        if e.status == 404:
            return False
        else:
            raise


def create_serviceaccount(body, namespace=None, timeout=K8S_CREATE_TIMEOUT):
    api = kube_client.CoreV1Api()
    name = body['metadata']['name']
    body["apiVersion"] = "v1"
    if namespace:
        body["metadata"]["namespace"] = namespace
    else:
        try:
            namespace = body["metadata"]["namespace"]
        except KeyError:
            namespace = "default"
    if not has_namespace(namespace):
        create_namespace(namespace)
    serviceaccount = api.create_namespaced_service_account(
        body=body,
        namespace=namespace)
    assert wait_for(p(has_serviceaccount, name, namespace=namespace), timeout_seconds=timeout), "timed out waiting for service account \"%s\" to be created!" % name
    return serviceaccount


def has_configmap(name, namespace="default"):
    api = kube_client.CoreV1Api()
    try:
        api.read_namespaced_config_map(name, namespace=namespace)
        return True
    except ApiException as e:
        if e.status == 404:
            return False
        else:
            raise


def create_configmap(body, namespace=None, timeout=K8S_CREATE_TIMEOUT):
    api = kube_client.CoreV1Api()
    name = body['metadata']['name']
    body["apiVersion"] = "v1"
    if namespace:
        body["metadata"]["namespace"] = namespace
    else:
        try:
            namespace = body["metadata"]["namespace"]
        except KeyError:
            namespace = "default"
    if not has_namespace(namespace):
        create_namespace(namespace)
    configmap = api.create_namespaced_config_map(
        body=body,
        namespace=namespace)
    assert wait_for(p(has_configmap, name, namespace=namespace), timeout_seconds=timeout), "timed out waiting for configmap \"%s\" to be created!" % name
    return configmap


def patch_configmap(body, namespace=None, timeout=K8S_CREATE_TIMEOUT):
    api = kube_client.CoreV1Api()
    name = body['metadata']['name']
    body["apiVersion"] = "v1"
    if namespace:
        body["metadata"]["namespace"] = namespace
    else:
        try:
            namespace = body["metadata"]["namespace"]
        except KeyError:
            namespace = "default"
    configmap = api.patch_namespaced_config_map(
        name=name,
        body=body,
        namespace=namespace)
    assert wait_for(p(has_configmap, name, namespace=namespace), timeout_seconds=timeout), "timed out waiting for configmap \"%s\" to be patched!" % name
    return configmap


def delete_configmap(name, namespace="default", timeout=K8S_DELETE_TIMEOUT):
    if not has_configmap(name, namespace=namespace):
        return
    api = kube_client.CoreV1Api()
    api.delete_namespaced_config_map(
        name=name,
        body=kube_client.V1DeleteOptions(grace_period_seconds=0, propagation_policy='Background'),
        namespace=namespace)
    assert wait_for(lambda: not has_configmap(name, namespace=namespace), timeout), "timed out waiting for configmap \"%s\" to be deleted!" % name


def has_deployment(name, namespace="default"):
    api = kube_client.ExtensionsV1beta1Api()
    try:
        api.read_namespaced_deployment(name, namespace=namespace)
        return True
    except ApiException as e:
        if e.status == 404:
            return False
        else:
            raise


def deployment_is_ready(name, replicas, namespace="default"):
    api = kube_client.ExtensionsV1beta1Api()
    if not has_deployment(name, namespace=namespace):
        return False
    if api.read_namespaced_deployment_status(name, namespace=namespace).status.ready_replicas == replicas:
        return True
    return False


def create_deployment(body, namespace=None, timeout=K8S_CREATE_TIMEOUT):
    api = kube_client.ExtensionsV1beta1Api()
    name = body["metadata"]["name"]
    body["apiVersion"] = "extensions/v1beta1"
    if namespace:
        body["metadata"]["namespace"] = namespace
    else:
        try:
            namespace = body["metadata"]["namespace"]
        except KeyError:
            namespace = "default"
    try:
        replicas = body["spec"]["replicas"]
    except KeyError:
        replicas = 1
        body["spec"]["replicas"] = replicas
    if not has_namespace(namespace):
        create_namespace(namespace)
    deployment = api.create_namespaced_deployment(
        body=body,
        namespace=namespace)
    assert wait_for(p(deployment_is_ready, name, replicas, namespace=namespace), timeout_seconds=timeout), "timed out waiting for deployment \"%s\" to be ready!" % name
    return deployment


def delete_deployment(name, namespace="default", timeout=K8S_DELETE_TIMEOUT):
    if not has_deployment(name, namespace=namespace):
        return
    api = kube_client.ExtensionsV1beta1Api()
    api.delete_namespaced_deployment(
        name=name,
        body=kube_client.V1DeleteOptions(grace_period_seconds=0, propagation_policy='Background'),
        namespace=namespace)
    assert wait_for(lambda: not has_deployment(name, namespace=namespace), timeout), "timed out waiting for deployment \"%s\" to be deleted!" % name


def has_daemonset(name, namespace="default"):
    api = kube_client.ExtensionsV1beta1Api()
    try:
        api.read_namespaced_daemon_set(name, namespace=namespace)
        return True
    except ApiException as e:
        if e.status == 404:
            return False
        else:
            raise


def daemonset_is_ready(name, namespace="default"):
    api = kube_client.ExtensionsV1beta1Api()
    if not has_daemonset(name, namespace=namespace):
        return False
    if api.read_namespaced_daemon_set_status(name, namespace=namespace).status.number_ready > 0:
        return True
    return False


def create_daemonset(body, namespace=None, timeout=K8S_CREATE_TIMEOUT):
    api = kube_client.ExtensionsV1beta1Api()
    name = body['metadata']['name']
    body["apiVersion"] = "extensions/v1beta1"
    if namespace:
        body["metadata"]["namespace"] = namespace
    else:
        try:
            namespace = body["metadata"]["namespace"]
        except KeyError:
            namespace = "default"
    if not has_namespace(namespace):
        create_namespace(namespace)
    daemonset = api.create_namespaced_daemon_set(
        body=body,
        namespace=namespace)
    assert wait_for(p(daemonset_is_ready, name, namespace=namespace), timeout_seconds=timeout), "timed out waiting for daemonset \"%s\" to be ready!" % name
    return daemonset


def delete_daemonset(name, namespace="default", timeout=K8S_DELETE_TIMEOUT):
    if not has_daemonset(name, namespace=namespace):
        return
    api = kube_client.ExtensionsV1beta1Api()
    api.delete_namespaced_daemon_set(
        name=name,
        body=kube_client.V1DeleteOptions(grace_period_seconds=0, propagation_policy='Background'),
        namespace=namespace)
    assert wait_for(lambda: not has_daemonset(name, namespace=namespace), timeout), "timed out waiting for daemonset \"%s\" to be deleted!" % name


def exec_pod_command(name, command, namespace="default"):
    api = kube_client.CoreV1Api()
    pods = get_all_pods_with_name(name, namespace=namespace)
    assert len(pods) == 1, "multiple pods found with name \"%s\"!\n%s" % (name, "\n".join([p.metadata.name for p in pods]))
    return api.connect_post_namespaced_pod_exec(
        name=pods[0].metadata.name,
        command=command,
        namespace=namespace)


def get_pod_logs(name, namespace="default"):
    api = kube_client.CoreV1Api()
    pods = get_all_pods_with_name(name, namespace=namespace)
    assert len(pods) == 1, "multiple pods found with name \"%s\"!\n%s" % (name, "\n".join([p.metadata.name for p in pods]))
    return api.read_namespaced_pod_log(
        name=pods[0].metadata.name,
        namespace=namespace)


def get_all_pods(namespace=None):
    """
    Args:
    namespace (str or None): Kubernetes namespace of pods

    Returns:
    If `namespace` is None, returns list of all pods in the cluster.
    Otherwise, returns list of pods within `namespace`.
    """
    api = kube_client.CoreV1Api()
    all_pods = api.list_pod_for_all_namespaces(watch=False)
    pods = []
    for pod in all_pods.items:
        if not namespace or pod.metadata.namespace == namespace:
            pods.append(pod)
    return pods


def get_all_pods_with_name(name, namespace=None):
    """
    Args:
    name (str):              Name of pods to search for
    namespace (str or None): Kubernetes namespace of pods

    Returns:
    If `namespace` is None, returns list of all pods in the cluster.
    Otherwise, returns list of pods within `namespace`.
    """
    pods = []
    for pod in get_all_pods(namespace=namespace):
        if name in pod.metadata.name:
            pods.append(pod)
    return pods


def has_pod(name, namespace=None):
    """
    Args:
    name (str):              Name of pods to search for
    namespace (str or None): Kubernetes namespace of pods


    Returns:
    If `namespace` is None, returns True/False if any pod contains `name`.
    Otherwise, returns True/False if any pod within `namespace` contains `name`.
    """
    for pod in get_all_pods(namespace=namespace):
        if name in pod.metadata.name:
            return True
    return False


def all_pods_have_ips(namespace="default"):
    """
    Args:
    namespace (str or None): Kubernetes namespace of pods

    Returns:
    If `namespace` is None, returns True/False if all pods have IPs.
    Otherwise, returns True/False if all pods in `namespace` have IPs.
    """
    pods = get_all_pods(namespace=namespace)
    if len(pods) == 0:
        return False
    ips = 0
    for pod in pods:
        if not pod.status.pod_ip:
            return False
        else:
            ips += 1
    if ips == len(pods):
        return True
    return False


def get_all_logs(minikube):
    """
    Args:
    minikube (Minikube): Minikube instance

    Returns:
    String containing:
    - the output from 'agent-status'
    - the agent container logs
    - the output from 'minikube logs'
    - the minikube container logs
    - the status of all pods
    """
    try:
        agent_status = minikube.agent.get_status()
    except:
        agent_status = ""
    try:
        agent_container_logs = minikube.agent.get_container_logs()
    except:
        agent_container_logs = ""
    try:
        _, output = minikube.container.exec_run("minikube logs")
        minikube_logs = output.decode('utf-8').strip()
    except:
        minikube_logs = ""
    try:
        minikube_container_logs = minikube.get_container_logs()
    except:
        minikube_container_logs = ""
    try:
        pods_status = ""
        for pod in get_all_pods():
            pods_status += "%s\t%s\t%s\n" % (pod.status.pod_ip, pod.metadata.namespace, pod.metadata.name)
        pods_status = pods_status.strip()
    except:
        pods_status = ""
    return "AGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n\nMINIKUBE LOGS:\n%s\n\nMINIKUBE CONTAINER LOGS:\n%s\n\nPODS STATUS:\n%s" % \
        (agent_status, agent_container_logs, minikube_logs, minikube_container_logs, pods_status)


def has_docker_image(client, name, tag=None):
    try:
        if tag:
            client.images.get(name + ":" + tag)
        else:
            client.images.get(name)
        return True
    except docker.errors.ImageNotFound:
        return False


def container_is_running(client, name):
    try:
        cont = client.containers.get(name)
        cont.reload()
        if cont.status.lower() != "running":
            return False
        return container_ip(cont)
    except docker.errors.NotFound:
        return False
    except docker.errors.APIError as e:
        if "is not running" in str(e):
            return False
        else:
            raise

def pod_port_open(container, host, port):
    rc, _ = container.exec_run("nc -z %s %d" % (host, port))
    return rc == 0


def get_free_port():
    s = socket.socket()
    s.bind(('', 0))
    return s.getsockname()[1]


def get_discovery_rule(yaml_file, observer, namespace="", container_index=0):
    """
    Generate container discovery rule based on yaml_file.

    Args:
    yaml_file (str):       Path to K8S deployment yaml.
    observer (str):        K8S observer type (e.g. k8s-api or k8s-kubelet).
    namespace (str):       K8S namespace.
    container_index (int): Index of the container in yaml_file to generate the discovery rule for.

    Returns:
    Discovery rule (str) that can be used for monitor configuration.
    The rule will include the following endpoint variables:
    - container_state (should always match "running")
    - discovered_by (should match the observer argument)
    - container_name
    - container_names (should include container_name)
    - container_image
    - container_labels with Contains and Get (if defined in the yaml_file pod spec)
    - port (if containerPort is defined in the yaml_file pod spec)
    - network_port (if containerPort is defined in the yaml_file pod_spec)
    - private_port (if containerPort is defined in the yaml_file pod_spec)
    - kubernetes_namespace (if the namespace argument is not empty or None)
    """
    name = None
    image = None
    ports = []
    labels = []
    with open(yaml_file, "r") as fd:
        for doc in yaml.load_all(fd.read()):
            if doc['kind'] == "Deployment":
                container = doc['spec']['template']['spec']['containers'][container_index]
                name = container['name']
                image = container['image']
                try:
                    ports = [p['containerPort'] for p in container['ports']]
                except KeyError:
                    ports = []
                try:
                    labels = doc['spec']['template']['metadata']['labels']
                except KeyError:
                    labels = []
    assert name, "failed to get container name from %s!" % yaml_file
    assert image, "failed to get container image from %s!" % yaml_file
    rule = 'container_state == "running"'
    rule += ' && discovered_by == "%s"' % observer
    rule += ' && container_name == "%s"' % name
    rule += ' && "%s" in container_names' % name
    rule += ' && container_image == "%s"' % image
    if len(labels) > 0:
        for key, value in labels.items():
            rule += ' && Contains(container_labels, "%s")' % key
            rule += ' && Get(container_labels, "%s") == "%s"' % (key, value)
    if len(ports) > 0:
        rule += ' && ((port == %s' % ports[0]
        rule += ' && network_port == %s' % ports[0]
        rule += ' && private_port == %s)' % ports[0]
        if len(ports) > 1:
            for port in ports[1:]:
                rule += ' || (port == %s' % port
                rule += ' && network_port == %s' % port
                rule += ' && private_port == %s)' % port
        rule += ")"
    if namespace:
        rule += ' && kubernetes_namespace == "%s"' % namespace
    return rule

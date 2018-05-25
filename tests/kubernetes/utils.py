from functools import partial as p
from kubernetes import client as kube_client
from kubernetes.client.rest import ApiException
from tests.helpers.assertions import *
from tests.helpers.util import *
import os
import netifaces as ni
import re
import time

K8S_API_TIMEOUT = 180


def get_metrics_from_doc(doc, ignore=[]):
    """
    Parse markdown document for metrics.

    Args:
    doc (str):            Path to markdown document
    ignore (list of str): Metrics to exclude/ignore

    Returns:
    Sorted set of metric names from `doc` excluding those in `ignore`
    """
    assert os.path.isfile(doc), "\"%s\" not found!" % doc
    with open(doc, 'r') as fd:
        metrics = set(re.findall('\|\s+`(.*?)`\s+\|\s+(?:counter|gauge|cumulative)\s+\|', fd.read(), re.IGNORECASE))
        if len(metrics) > 0 and len(ignore) > 0:
            metrics.difference_update(set(ignore))
        return set(sorted(metrics))


# returns a sorted set of dimension names from `doc` excluding those in `ignore`
def get_dims_from_doc(doc, ignore=[]):
    """
    Parse markdown document for dimensions.

    Args:
    doc (str):            Path to markdown document
    ignore (list of str): Dimensions to exclude/ignore

    Returns:
    Sorted set of dimensions from `doc` excluding those in `ignore`
    """
    assert os.path.isfile(doc), "\"%s\" not found!" % doc
    with open(doc, 'r') as fd:
        dims = set()
        line = fd.readline()
        while line and not re.match('\s*##\s*Dimensions.*', line):
            line = fd.readline()
        if line:
            dim_line = re.compile('\|\s+`(.*?)`\s+\|.*\|')
            match = None
            while line and not match:
                line = fd.readline()
                match = dim_line.match(line)
            while line and match:
                dims.add(match.group(1))
                line = fd.readline()
                match = dim_line.match(line)
        if len(dims) > 0 and len(ignore) > 0:
            dims.differnce_update(set(ignore))
        return set(sorted(dims))


def get_host_ip():
    gws = ni.gateways()
    interface = gws['default'][ni.AF_INET][1]
    return ni.ifaddresses(interface)[ni.AF_INET][0]['addr']


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
    api = kube_client.CoreV1Api()
    return api.create_namespaced_secret(
        body=kube_client.V1Secret(
            metadata=kube_client.V1ObjectMeta(name=name),
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


def create_serviceaccount(body=None, namespace="default", timeout=K8S_API_TIMEOUT):
    api = kube_client.CoreV1Api()
    if body:
        name = body['metadata']['name']
        body["apiVersion"] = "v1"
        try:
            namespace = body["metadata"]["namespace"]
        except:
            body["metadata"]["namespace"] = namespace
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


def create_configmap(body=None, name="", data={}, labels={}, namespace="default", timeout=K8S_API_TIMEOUT):
    api = kube_client.CoreV1Api()
    if body:
        name = body['metadata']['name']
        body["apiVersion"] = "v1"
        try:
            namespace = body["metadata"]["namespace"]
        except:
            body["metadata"]["namespace"] = namespace
    else:
        body = kube_client.V1ConfigMap(
            api_version="v1",
            kind="ConfigMap",
            metadata=kube_client.V1ObjectMeta(name=name, labels=labels),
            data=data)
    configmap = api.create_namespaced_config_map(
        body=body,
        namespace=namespace)
    assert wait_for(p(has_configmap, name, namespace=namespace), timeout_seconds=timeout), "timed out waiting for configmap \"%s\" to be created!" % name
    return configmap


def patch_configmap(body, namespace="default", timeout=K8S_API_TIMEOUT):
    api = kube_client.CoreV1Api()
    name = body['metadata']['name']
    body["apiVersion"] = "v1"
    try:
        namespace = body["metadata"]["namespace"]
    except:
        body["metadata"]["namespace"] = namespace
    configmap =  api.patch_namespaced_config_map(
        name=name,
        body=body,
        namespace=namespace)
    assert wait_for(p(has_configmap, name, namespace=namespace), timeout_seconds=timeout), "timed out waiting for configmap \"%s\" to be patched!" % name
    return configmap


def delete_configmap(name, namespace="default", timeout=K8S_API_TIMEOUT):
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


def create_deployment(body=None, name="", pod_template=None, replicas=1, labels={}, namespace="default", timeout=K8S_API_TIMEOUT):
    api = kube_client.ExtensionsV1beta1Api()
    if body:
        name = body["metadata"]["name"]
        body["apiVersion"] = "extensions/v1beta1"
        try:
            namespace = body["metadata"]["namespace"]
        except:
            body["metadata"]["namespace"] = namespace
    else:
        spec = kube_client.ExtensionsV1beta1DeploymentSpec(
            replicas=replicas,
            template=pod_template)
        body = kube_client.ExtensionsV1beta1Deployment(
            api_version="extensions/v1beta1",
            kind="Deployment",
            metadata=kube_client.V1ObjectMeta(name=name, labels=labels),
            spec=spec)
    deployment = api.create_namespaced_deployment(
        body=body,
        namespace=namespace)
    assert wait_for(p(has_deployment, name, namespace=namespace), timeout_seconds=timeout), "timed out waiting for deployment \"%s\" to be created!" % name
    return deployment


def delete_deployment(name, namespace="default", timeout=K8S_API_TIMEOUT):
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


def create_daemonset(body=None, namespace="default", timeout=K8S_API_TIMEOUT):
    api = kube_client.ExtensionsV1beta1Api()
    if body:
        name = body['metadata']['name']
        body["apiVersion"] = "extensions/v1beta1"
        try:
            namespace = body["metadata"]["namespace"]
        except:
            body["metadata"]["namespace"] = namespace
    daemonset = api.create_namespaced_daemon_set(
        body=body,
        namespace=namespace)
    assert wait_for(p(has_daemonset, name, namespace=namespace), timeout_seconds=timeout), "timed out waiting for daemonset \"%s\" to be created!" % name
    return daemonset


def delete_daemonset(name, namespace="default", timeout=K8S_API_TIMEOUT):
    if not has_daemonset(name, namespace=namespace):
        return
    api = kube_client.ExtensionsV1beta1Api()
    api.delete_namespaced_daemon_set(
        name=name,
        body=kube_client.V1DeleteOptions(grace_period_seconds=0, propagation_policy='Background'),
        namespace=namespace)
    assert wait_for(lambda: not has_daemonset(name, namespace=namespace), timeout), "timed out waiting for daemonset \"%s\" to be deleted!" % name


def get_all_pods():
    """
    Returns:
    List of all pods in the cluster
    """
    v1 = kube_client.CoreV1Api()
    pods = v1.list_pod_for_all_namespaces(watch=False)
    return pods.items


def get_all_pods_matching_name(name):
    """
    Args:
    name (str or regex): Name of pod(s) to search for

    Returns:
    List of all pods in the cluster that matches `name`
    """
    pods = []
    for pod in get_all_pods():
        if re.match(name, pod.metadata.name):
            pods.append(pod)
    return pods


def has_pod(name):
    """
    Args:
    name (str): Name of pod(s) to search for

    Returns:
    True if any pod contains `pod_name`; otherwise False
    """
    for pod in get_all_pods():
        if name in pod.metadata.name:
            return True
    return False


def all_pods_have_ips():
    """
    Returns:
    True if all pods have IPs; otherwise False
    """
    pods = get_all_pods()
    if len(pods) == 0:
        return False
    ips = 0
    for pod in pods:
        if not pod.status.pod_ip:
            return False
        else:
            ips += 1
    if ips == len(pods):
        for pod in pods:
            print("%s\t%s\t%s" % (pod.status.pod_ip, pod.metadata.namespace, pod.metadata.name))
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

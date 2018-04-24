from kubernetes import (
    client as kube_client,
    watch as kube_watch
)
from contextlib import contextmanager
from functools import partial as p
from tests.helpers.assertions import *
from tests.helpers.util import *

import os
import netifaces as ni
import re
import sys
import time
import yaml

DOCS_DIR = os.environ.get("DOCS_DIR", "/go/src/github.com/signalfx/signalfx-agent/docs/monitors")

# check the fake backend for a list of metrics
# returns a list of the metrics not found if the timeout is reached
# returns an empty list if all metrics are found before the timeout is reached
def check_for_metrics(backend, metrics, timeout):
    start_time = time.time()
    while True:
        if (time.time() - start_time) > timeout:
            break
        for metric in metrics:
            if has_datapoint_with_metric_name(backend, metric):
                metrics.remove(metric)
        if len(metrics) == 0:
            break
        time.sleep(5)
    return metrics

# check the fake backend for a list of dimensions
# returns a list of the dimensions not found if the timeout is reached
# returns an empty list if all dimensions are found before the timeout is reached
def check_for_dims(backend, dims, timeout):
    start_time = time.time()
    while True:
        if (time.time() - start_time) > timeout:
            break
        for dim in dims:
            if not dim["value"] or has_datapoint_with_dim(backend, dim["key"], dim["value"]):
                dims.remove(dim)
        if len(dims) == 0:
            break
        time.sleep(5)
    return dims

# returns a sorted list of unique metric names from `doc` excluding those in `ignore`
def get_metrics_from_doc(doc, ignore=[]):
    metrics = []
    if not os.path.isfile(doc):
        doc = os.path.join(DOCS_DIR, doc)
    assert os.path.isfile(doc), "\"%s\" not found!" % doc
    with open(doc) as fd:
        metrics = re.findall('\|\s+`(.*?)`\s+\|\s+(?:counter|gauge|cumulative)\s+\|', fd.read(), re.IGNORECASE)
    if len(ignore) > 0:
        metrics = [i for i in metrics if i not in ignore]
    return sorted(list(set(metrics)))

# returns a sorted list of unique dimension names from `doc` excluding those in `ignore`
def get_dims_from_doc(doc, ignore=[]):
    dims = []
    if not os.path.isfile(doc):
        doc = os.path.join(DOCS_DIR, doc)
    assert os.path.isfile(doc), "\"%s\" not found!" % doc
    with open(doc) as fd:
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
                dims.append(match.group(1))
                line = fd.readline()
                match = dim_line.match(line)
    if len(ignore) > 0:
        dims = [i for i in dims if i not in ignore]
    return sorted(list(set(dims)))

# returns the IP of the pytest host (i.e. the dev image)
def get_host_ip():
    return ni.ifaddresses('eth0')[ni.AF_INET][0]['addr']

def create_configmap(name="", body=None, data={}, labels={}, namespace="default"):
    v1 = kube_client.CoreV1Api()
    if not body and name and data:
        body = kube_client.V1ConfigMap(
            api_version="v1",
            kind="ConfigMap",
            metadata=kube_client.V1ObjectMeta(name=name, labels=labels),
            data=data)
    return v1.create_namespaced_config_map(
        body=body,
        namespace=namespace)

def get_pod_template(name="", image="", port=None, labels={}, volume_mounts=[]):
    def get_volume_mounts():
        mounts = []
        for vm in volume_mounts:
            mounts.append(kube_client.V1VolumeMount(name=vm["name"], mount_path=vm["mount_path"]))
        return mounts

    def get_configmap_volumes():
        configmap_volumes = []
        for vm in volume_mounts:
            configmap_volumes.append(kube_client.V1Volume(name=vm["name"], config_map=kube_client.V1ConfigMapVolumeSource(name=vm["configmap"])))
        return configmap_volumes

    container = kube_client.V1Container(
        name=name,
        image=image,
        ports=[kube_client.V1ContainerPort(container_port=port)],
        volume_mounts=get_volume_mounts())
    template = kube_client.V1PodTemplateSpec(
        metadata=kube_client.V1ObjectMeta(labels=labels),
        spec=kube_client.V1PodSpec(
            containers=[container],
            volumes=get_configmap_volumes()))
    return template

def create_deployment(name="", pod_template=None, replicas=1, labels={}, namespace="default"):
    v1beta1 = kube_client.ExtensionsV1beta1Api()
    spec = kube_client.ExtensionsV1beta1DeploymentSpec(
        replicas=replicas,
        template=pod_template)
    deployment = kube_client.ExtensionsV1beta1Deployment(
        api_version="extensions/v1beta1",
        kind="Deployment",
        metadata=kube_client.V1ObjectMeta(name=name, labels=labels),
        spec=spec)
    return v1beta1.create_namespaced_deployment(
        body=deployment,
        namespace=namespace)

def create_replication_controller(name="", pod_template=None, replicas=1, labels={}, namespace="default"):
    v1 = kube_client.CoreV1Api()
    spec = kube_client.V1ReplicationControllerSpec(
        replicas=replicas,
        template=pod_template,
        selector=labels)
    rc = kube_client.V1ReplicationController(
        api_version="v1",
        metadata=kube_client.V1ObjectMeta(name=name, labels=labels),
        spec=spec)
    return v1.create_namespaced_replication_controller(
        body=rc,
        namespace=namespace)

def create_service(name="", port=None, service_type="NodePort", labels={}, namespace="default"):
    v1 = kube_client.CoreV1Api()
    service = kube_client.V1Service(
        api_version="v1",
        kind="Service",
        metadata=kube_client.V1ObjectMeta(name=name, labels=labels),
        spec=kube_client.V1ServiceSpec(
            type=service_type,
            ports=[kube_client.V1ServicePort(port=port)],
            selector=labels))
    return v1.create_namespaced_service(
        body=service,
        namespace=namespace)

def create_daemonset(body=None, namespace="default"):
    v1beta1 = kube_client.ExtensionsV1beta1Api()
    return v1beta1.create_namespaced_daemon_set(
        body=body,
        namespace=namespace)

def create_serviceaccount(body=None, namespace="default"):
    v1 = kube_client.CoreV1Api()
    return v1.create_namespaced_service_account(
        body=body,
        namespace=namespace)

# returns a list of all pods in the cluster
def get_all_pods():
    v1 = kube_client.CoreV1Api()
    pods = v1.list_pod_for_all_namespaces(watch=False)
    return pods.items

# returns a list of all pods in the cluster that regex matches `name`
def get_all_pods_matching_name(name):
    pods = []
    for pod in get_all_pods():
        if re.match(name, pod.metadata.name):
            pods.append(pod)
    return pods

# returns True if any pod contains `pod_name`; False otherwise
def has_pod(pod_name):
    for pod in get_all_pods():
        if pod_name in pod.metadata.name:
            return True
    return False

# returns True if all pods have IPs; False otherwise
def all_pods_have_ips():
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

# returns a string containing:
# - the output from 'agent-status'
# - the agent container logs
# - the output from 'minikube logs'
# - the minikube container logs
# - the status of all pods
def get_all_logs(minikube):
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


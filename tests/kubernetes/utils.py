from functools import partial as p
from kubernetes import client as kube_client
from tests.helpers.assertions import *
from tests.helpers.util import *
import os
import netifaces as ni
import re
import time

DOCS_DIR = os.environ.get("DOCS_DIR", "/go/src/github.com/signalfx/signalfx-agent/docs/monitors")
K8S_API_TIMEOUT = 120

# check the fake backend if any metric in `metrics` exist
# returns the datapoint if found before the timeout is reached, otherwise None
def any_metric_found(backend, metrics, timeout):
    start_time = time.time()
    while True:
        if (time.time() - start_time) > timeout:
            return None
        for dp in backend.datapoints:
            if dp.metric in metrics:
                return dp
        time.sleep(2)


# check the fake backend if any dimension in `dims` exist
# returns the datapoint if found before the timeout is reached, otherwise None
def any_dim_found(backend, dims, timeout):
    start_time = time.time()
    while True:
        if (time.time() - start_time) > timeout:
            return None
        for dp in backend.datapoints:
            for dim in dp.dimensions:
                if dim.key in dims:
                    return dp
        time.sleep(2)


# check the fake backend if any metric in `metrics` with any dimension in `dims` exist
# returns the datapoint if found before the timeout is reached, otherwise None
def any_metric_has_any_dim(backend, metrics, dims, timeout):
    start_time = time.time()
    while True:
        if (time.time() - start_time) > timeout:
            return None
        for dp in backend.datapoints:
            if dp.metric in metrics:
                for dim in dp.dimensions:
                    if dim.key in dims:
                        return dp
        time.sleep(2)


# check the fake backend for a list of metrics
# returns a list of the metrics not found if the timeout is reached
# returns an empty list if all metrics are found before the timeout is reached
def check_for_metrics(backend, metrics, timeout):
    metrics_not_found = list(metrics)
    start_time = time.time()
    while True:
        if (time.time() - start_time) > timeout:
            break
        for metric in metrics_not_found:
            if has_datapoint_with_metric_name(backend, metric):
                metrics_not_found.remove(metric)
        if len(metrics_not_found) == 0:
            break
        time.sleep(5)
    return metrics_not_found


# check the fake backend for a list of dimensions
# returns a list of the dimensions not found if the timeout is reached
# returns an empty list if all dimensions are found before the timeout is reached
def check_for_dims(backend, dims, timeout):
    dims_not_found = list(dims)
    start_time = time.time()
    while True:
        if (time.time() - start_time) > timeout:
            break
        for dim in dims_not_found:
            if not dim["value"] or has_datapoint_with_dim(backend, dim["key"], dim["value"]):
                dims_not_found.remove(dim)
        if len(dims_not_found) == 0:
            break
        time.sleep(5)
    return dims_not_found


# returns a sorted list of unique metric names from `doc` excluding those in `ignore`
def get_metrics_from_doc(doc, ignore=[]):
    if not os.path.isfile(doc):
        doc = os.path.join(DOCS_DIR, doc)
    assert os.path.isfile(doc), "\"%s\" not found!" % doc
    with open(doc) as fd:
        all_metrics = re.findall('\|\s+`(.*?)`\s+\|\s+(?:counter|gauge|cumulative)\s+\|', fd.read(), re.IGNORECASE)
        metrics = list(all_metrics)
    if len(ignore) > 0:
        for m in all_metrics:
            for i in ignore:
                if re.match(i, m):
                    metrics.remove(m)
    return sorted(list(set(metrics)))


# returns a sorted list of unique dimension names from `doc` excluding those in `ignore`
def get_dims_from_doc(doc, ignore=[]):
    if not os.path.isfile(doc):
        doc = os.path.join(DOCS_DIR, doc)
    assert os.path.isfile(doc), "\"%s\" not found!" % doc
    with open(doc) as fd:
        all_dims = []
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
                all_dims.append(match.group(1))
                line = fd.readline()
                match = dim_line.match(line)
        dims = list(all_dims)
    if len(ignore) > 0:
        for d in all_dims:
            for i in ignore:
                if re.match(i, d):
                    dims.remove(d)
    return sorted(list(set(dims)))


# returns the IP of the pytest host (i.e. the dev image)
def get_host_ip():
    return ni.ifaddresses('eth0')[ni.AF_INET][0]['addr']


def has_serviceaccount(name, namespace="default"):
    api = kube_client.CoreV1Api()
    service_accounts = api.list_namespaced_service_account(namespace=namespace)
    if service_accounts:
        for sa in service_accounts.items:
            if sa.metadata.name == name:
                return True
    return False


def create_serviceaccount(body=None, namespace="default", timeout=K8S_API_TIMEOUT):
    api = kube_client.CoreV1Api()
    if body:
        name = body['metadata']['name']
        body["apiVersion"] = "v1"
        try:
            namespace = body["metadata"]["namespace"]
        except:
            pass
    serviceaccount = api.create_namespaced_service_account(
        body=body,
        namespace=namespace)
    assert wait_for(p(has_serviceaccount, name, namespace=namespace), timeout_seconds=timeout), "timed out waiting for service account \"%s\" to be created!" % name
    return serviceaccount


def has_configmap(name, namespace="default"):
    api = kube_client.CoreV1Api()
    configmaps = api.list_namespaced_config_map(namespace=namespace)
    if configmaps:
        for cm in configmaps.items:
            if cm.metadata.name == name:
                return True
    return False


def create_configmap(body=None, name="", data={}, labels={}, namespace="default", timeout=K8S_API_TIMEOUT):
    api = kube_client.CoreV1Api()
    if body:
        name = body['metadata']['name']
        body["apiVersion"] = "v1"
        try:
            namespace = body["metadata"]["namespace"]
        except:
            pass
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


def delete_configmap(name, namespace="default", timeout=K8S_API_TIMEOUT):
    if not has_configmap(name, namespace=namespace):
        return
    api = kube_client.CoreV1Api()
    api.delete_namespaced_config_map(
        name=name,
        body=kube_client.V1DeleteOptions(grace_period_seconds=0, propagation_policy='Foreground'),
        namespace=namespace)
    start_time = time.time()
    while True:
        assert (time.time() - start_time) <= timeout, "timed out waiting for configmap \"%s\" to be deleted!" % name
        if has_configmap(name, namespace=namespace):
            time.sleep(2)
        else:
            break


def get_pod_template(name="", image="", ports=[], labels={}, volume_mounts=[], env={}, command=[], args=[]):
    def get_ports():
        container_ports = []
        for port in ports:
            container_ports.append(kube_client.V1ContainerPort(container_port=port))
        return container_ports

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

    def get_envvars():
        envvars = []
        for key,val in env.items():
            envvars.append(kube_client.V1EnvVar(name=key, value=val))
        return envvars

    container = kube_client.V1Container(
        name=name,
        image=image,
        ports=get_ports(),
        volume_mounts=get_volume_mounts(),
        env=get_envvars(),
        command=command,
        args=args)
    template = kube_client.V1PodTemplateSpec(
        metadata=kube_client.V1ObjectMeta(labels=labels),
        spec=kube_client.V1PodSpec(
            containers=[container],
            volumes=get_configmap_volumes()))
    return template


def has_deployment(name, namespace="default"):
    api = kube_client.ExtensionsV1beta1Api()
    deployments = api.list_namespaced_deployment(namespace=namespace)
    if deployments:
        for d in deployments.items:
            if d.metadata.name == name:
                return True
    return False


def create_deployment(body=None, name="", pod_template=None, replicas=1, labels={}, namespace="default", timeout=K8S_API_TIMEOUT):
    api = kube_client.ExtensionsV1beta1Api()
    if body:
        name = body["metadata"]["name"]
        body["apiVersion"] = "extensions/v1beta1"
        try:
            namespace = body["metadata"]["namespace"]
        except:
            pass
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
        body=kube_client.V1DeleteOptions(grace_period_seconds=0, propagation_policy='Foreground'),
        namespace=namespace)
    start_time = time.time()
    while True:
        assert (time.time() - start_time) <= timeout, "timed out waiting for deployment \"%s\" to be deleted!" % name
        if has_deployment(name, namespace=namespace):
            time.sleep(2)
        else:
            break


def create_replication_controller(body=None, name="", pod_template=None, replicas=1, labels={}, namespace="default"):
    api = kube_client.CoreV1Api()
    if body:
        body["apiVersion"] = "v1"
        try:
            namespace = body["metadata"]["namespace"]
        except:
            pass
    else:
        spec = kube_client.V1ReplicationControllerSpec(
            replicas=replicas,
            template=pod_template,
            selector=labels)
        body = kube_client.V1ReplicationController(
            api_version="v1",
            metadata=kube_client.V1ObjectMeta(name=name, labels=labels),
            spec=spec)
    return api.create_namespaced_replication_controller(
        body=body,
        namespace=namespace)


def create_service(name="", ports=[], service_type="NodePort", labels={}, namespace="default"):
    def get_ports():
        service_ports = []
        for port in ports:
            service_ports.append(kube_client.V1ServicePort(port=port))
        return service_ports

    api = kube_client.CoreV1Api()
    service = kube_client.V1Service(
        api_version="v1",
        kind="Service",
        metadata=kube_client.V1ObjectMeta(name=name, labels=labels),
        spec=kube_client.V1ServiceSpec(
            type=service_type,
            ports=get_ports(),
            selector=labels))
    return api.create_namespaced_service(
        body=service,
        namespace=namespace)


def has_daemonset(name, namespace="default"):
    api = kube_client.ExtensionsV1beta1Api()
    daemonsets = api.list_namespaced_daemon_set(namespace=namespace)
    if daemonsets:
        for ds in daemonsets.items:
            if ds.metadata.name == name:
                return True
    return False


def create_daemonset(body=None, namespace="default", timeout=K8S_API_TIMEOUT):
    api = kube_client.ExtensionsV1beta1Api()
    if body:
        name = body['metadata']['name']
        body["apiVersion"] = "extensions/v1beta1"
        try:
            namespace = body["metadata"]["namespace"]
        except:
            pass
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
        body=kube_client.V1DeleteOptions(grace_period_seconds=0, propagation_policy='Foreground'),
        namespace=namespace)
    start_time = time.time()
    while True:
        assert (time.time() - start_time) <= timeout, "timed out waiting for daemonset \"%s\" to be deleted!" % name
        if has_daemonset(name, namespace=namespace):
            time.sleep(2)
        else:
            break


def deploy_k8s_service(**kwargs):
    if configmap_name and configmap_data:
        create_configmap(
            name=configmap_name,
            data=configmap_data,
            labels=labels,
            namespace=namespace)
    pod_template = get_pod_template(
        name=name,
        image=image,
        ports=ports,
        labels=labels,
        env=env,
        command=command,
        args=args,
        volume_mounts=volume_mounts)
    if replicas > 1:
        create_replication_controller(
            name=pod_name,
            pod_template=pod_template,
            replicas=replicas,
            labels=labels,
            namespace=namespace)
    else:
        create_deployment(
            name=pod_name,
            pod_template=pod_template,
            replicas=replicas,
            labels=labels,
            namespace=namespace)
    if service_type:
        create_service(
            name="%s-service" % name,
            ports=ports,
            service_type=service_type,
            labels=labels,
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

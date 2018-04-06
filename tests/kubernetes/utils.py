from kubernetes import (
    client as kube_client,
    watch as kube_watch
)
from tests.helpers.util import *

import os
import netifaces as ni
import time

def env_is_circleci():
    if os.environ.get("CIRCLECI") and os.environ.get("CIRCLECI") == "true":
        return True
    return False

def get_host_ip():
    #proc = subprocess.run("ip r | awk '/default/{print $7}'", shell=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    #ip = proc.stdout.decode('utf-8').strip()
    #assert ip != "", "failed to get system IP!"
    #assert re.match('(\d{1,3}\.){3}\d{1,3}', ip), "failed to get system IP!\n%s" % proc.stdout.decode('utf-8').strip()
    return ni.ifaddresses('eth0')[ni.AF_INET][0]['addr']

def pull_agent_image(container, client, image_name="", image_tag=""):
    container.exec_run("cp -f /etc/hosts /etc/hosts.orig")
    container.exec_run("cp -f /etc/hosts /etc/hosts.new")
    container.exec_run("sed -i 's|127.0.0.1|%s|' /etc/hosts.new" % get_host_ip())
    container.exec_run("cp -f /etc/hosts.new /etc/hosts")
    time.sleep(5)
    client.images.pull(image_name, tag=image_tag)
    container.exec_run("cp -f /etc/hosts.orig /etc/hosts")
    _, output = container.exec_run('docker images')
    print_lines(output.decode('utf-8'))

def get_kubeconfig(container, kubeconfig_path="/kubeconfig"):
    rc, output = container.exec_run("cp -f %s /tmp/scratch/kubeconfig-%s" % (kubeconfig_path, container.id[:12]))
    assert rc == 0, "failed to get %s from minikube!\n%s" % (kubeconfig_path, output.decode('utf-8'))
    return "/tmp/scratch/kubeconfig-%s" % container.id[:12]

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
    def get_volume_mounts(volume_mounts):
        mounts = []
        for vm in volume_mounts:
            mounts.append(kube_client.V1VolumeMount(name=vm["name"], mount_path=vm["mount_path"]))
        return mounts

    def get_configmap_volumes(volumes_mounts):
        configmap_volumes = []
        for vm in volume_mounts:
            configmap_volumes.append(kube_client.V1Volume(name=vm["name"], config_map=kube_client.V1ConfigMapVolumeSource(name=vm["configmap"])))
        return configmap_volumes

    container = kube_client.V1Container(
        name=name,
        image=image,
        ports=[kube_client.V1ContainerPort(container_port=port)],
        volume_mounts=get_volume_mounts(volume_mounts))
    template = kube_client.V1PodTemplateSpec(
        metadata=kube_client.V1ObjectMeta(labels=labels),
        spec=kube_client.V1PodSpec(
            containers=[container],
            volumes=get_configmap_volumes(volume_mounts)))
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

def create_deployment_from_yaml(yaml_path):
    with open(yaml_path) as f:
        dep = yaml.load(f)
        k8s_beta = kube_client.ExtensionsV1beta1Api()
        resp = k8s_beta.create_namespaced_deployment(
            body=dep, namespace="default")
        print("Deployment created. status='%s'" % str(resp.status))

def update_deployment(api_instance, deployment):
    # Update container image
    deployment.spec.template.spec.containers[0].image = "nginx:latest"
    # Update the deployment
    api_response = api_instance.patch_namespaced_deployment(
        name=DEPLOYMENT_NAME,
        namespace="default",
        body=deployment)
    print("Deployment updated. status='%s'" % str(api_response.status))

def delete_deployment(api_instance):
    # Delete deployment
    api_response = api_instance.delete_namespaced_deployment(
        name=DEPLOYMENT_NAME,
        namespace="default",
        body=client.V1DeleteOptions(
            propagation_policy='Foreground',
            grace_period_seconds=5))
    print("Deployment deleted. status='%s'" % str(api_response.status))

def has_pod(pod_name):
    v1 = kube_client.CoreV1Api()
    pods = v1.list_pod_for_all_namespaces(watch=False)
    for pod in pods.items:
        if pod_name in pod.metadata.name:
            return True
    return False

def all_pods_have_ips():
    v1 = kube_client.CoreV1Api()
    pods = v1.list_pod_for_all_namespaces(watch=False)
    if len(pods.items) == 0:
        return False
    ips = 0
    for pod in pods.items:
        if not pod.status.pod_ip:
            return False
        else:
            ips += 1
    if ips == len(pods.items):
        for pod in pods.items:
            print("%s\t%s\t%s" % (pod.status.pod_ip, pod.metadata.namespace, pod.metadata.name))
        return True
    return False

def get_agent_container(client, image_name="", image_tag="", timeout=60):
    start_time = time.time()
    while True:
        if (time.time() - start_time) > timeout:
            return None
        try:
            return client.containers.list(filters={"ancestor": image_name + ":" + image_tag})[0]
        except:
            time.sleep(1)

def get_agent_status(agent_container):
    try:
        rc, output = agent_container.exec_run("agent-status")
        if rc != 0:
            raise Exception(output.decode('utf-8').strip())
        return output.decode('utf-8').strip()
    except Exception as e:
        return "Failed to get agent-status!\n%s" % str(e)

def get_agent_container_logs(agent_container):
    try:
        return agent_container.logs().decode('utf-8').strip()
    except Exception as e:
        return "Failed to get agent container logs!\n%s" % str(e)

def get_all_logs(agent_container, minikube_container):
    try:
        agent_status = get_agent_status(agent_container)
    except:
        agent_status = ""
    try:
        agent_container_logs = get_agent_container_logs(agent_container)
    except:
        agent_container_logs = ""
    try:
        _, output = minikube_container.exec_run("minikube logs")
        minikube_logs = output.decode('utf-8').strip()
    except:
        minikube_logs = ""
    try:
        pods_status = ""
        v1 = kube_client.CoreV1Api()
        pods = v1.list_pod_for_all_namespaces(watch=False)
        for pod in pods.items:
            pods_status += "%s\t%s\t%s\n" % (pod.status.pod_ip, pod.metadata.namespace, pod.metadata.name)
        pods_status = pods_status.strip()
    except:
        pods_status = ""
    return "AGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n\nMINIKUBE LOGS:\n%s\n\nPODS STATUS:\n%s" % \
        (agent_status, agent_container_logs, minikube_logs, pods_status)

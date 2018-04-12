from kubernetes import (
    client as kube_client,
    config as kube_config,
    watch as kube_watch
)
from contextlib import contextmanager
from functools import partial as p
from tests.helpers.assertions import *
from tests.helpers.util import *

import docker
import os
import netifaces as ni
import pytest
import re
import sys
import time
import yaml

AGENT_YAMLS_DIR = os.environ.get("AGENT_YAMLS_DIR", "/go/src/github.com/signalfx/signalfx-agent/deployments/k8s")
AGENT_CONFIGMAP_PATH = os.environ.get("AGENT_CONFIGMAP_PATH", os.path.join(AGENT_YAMLS_DIR, "configmap.yaml"))
AGENT_DAEMONSET_PATH = os.environ.get("AGENT_DAEMONSET_PATH", os.path.join(AGENT_YAMLS_DIR, "daemonset.yaml"))
AGENT_SERVICEACCOUNT_PATH = os.environ.get("AGENT_SERVICEACCOUNT_PATH", os.path.join(AGENT_YAMLS_DIR, "serviceaccount.yaml"))
AGENT_IMAGE_NAME = "localhost:5000/signalfx-agent"
AGENT_IMAGE_TAG = "k8s-test"
MINIKUBE_VERSION = os.environ.get("MINIKUBE_VERSION", "latest")
SERVICES_DIR = os.path.join(os.path.dirname(os.path.realpath(__file__)), 'services')
SERVICES = []
sys.path.append(SERVICES_DIR)
for service in os.listdir(SERVICES_DIR):
    if service != '__init__.py' and service.endswith('.py'):
        exec("import %s" % service[:-3])
        SERVICES.append(service[:-3])

def get_metrics_from_docs(docs=[], ignored_metrics=[]):
    all_metrics = []
    for doc in docs:
        with open(doc) as fd:
            metrics = re.findall('\|\s+`(.*?)`\s+\|\s+(?:counter|gauge|cumulative)\s+\|', fd.read(), re.IGNORECASE)
            assert len(metrics) > 0, "Failed to get metrics from %s!" % doc
            all_metrics += metrics
    if len(ignored_metrics) > 0:
        all_metrics = [i for i in all_metrics if i not in ignored_metrics]
    return sorted(list(set(all_metrics)))

@pytest.fixture(scope="session")
def local_registry(request):
    client = docker.from_env(version='auto')
    final_agent_image_name = request.config.getoption("--k8s-agent-name")
    final_agent_image_tag = request.config.getoption("--k8s-agent-tag")
    try:
        final_image = client.images.get(final_agent_image_name + ":" + final_agent_image_tag)
    except:
        try:
            print("\nAgent image '%s:%s' not found in local registry." % (final_agent_image_name, final_agent_image_tag))
            print("\nAttempting to pull from remote registry ...")
            final_image = client.images.pull(final_agent_image_name, tag=final_agent_image_tag)
        except:
            final_image = None
    assert final_image, "agent image '%s:%s' not found!" % (final_agent_image_name, final_agent_image_tag)
    try:
        client.containers.get("registry")
        print("\nRegistry container localhost:5000 already running")
    except:
        try:
            client.containers.run(
                image='registry:latest',
                name='registry',
                remove=True,
                detach=True,
                ports={'5000/tcp': 5000})
            print("\nStarted registry container localhost:5000")
        except:
            pass
        print("\nWaiting for registry container localhost:5000 to be ready ...")
        start_time = time.time()
        while True:
            assert (time.time() - start_time) < 30, "timed out waiting for registry container to be ready!"
            try:
                client.containers.get("registry")
                time.sleep(2)
                break
            except:
                time.sleep(2)
    print("\nTagging %s:%s as %s:%s ..." % (final_agent_image_name, final_agent_image_tag, AGENT_IMAGE_NAME, AGENT_IMAGE_TAG))
    final_image.tag(AGENT_IMAGE_NAME, tag=AGENT_IMAGE_TAG)
    print("\nPushing %s:%s ..." % (AGENT_IMAGE_NAME, AGENT_IMAGE_TAG))
    client.images.push(AGENT_IMAGE_NAME, tag=AGENT_IMAGE_TAG)

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
    time.sleep(2)
    rc, output = container.exec_run("cp -f %s /tmp/scratch/kubeconfig-%s" % (kubeconfig_path, container.id[:12]))
    assert rc == 0, "failed to get %s from minikube!\n%s" % (kubeconfig_path, output.decode('utf-8'))
    time.sleep(2)
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

def get_all_pods():
    v1 = kube_client.CoreV1Api()
    pods = v1.list_pod_for_all_namespaces(watch=False)
    return pods.items

def get_all_pods_with_name(name):
    pods = []
    for pod in get_all_pods():
        if re.search(name, pod.metadata.name):
            pods.append(pod)
    return pods

def has_pod(pod_name):
    for pod in get_all_pods():
        if pod_name in pod.metadata.name:
            return True
    return False

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
        for pod in get_all_pods():
            pods_status += "%s\t%s\t%s\n" % (pod.status.pod_ip, pod.metadata.namespace, pod.metadata.name)
        pods_status = pods_status.strip()
    except:
        pods_status = ""
    return "AGENT STATUS:\n%s\n\nAGENT CONTAINER LOGS:\n%s\n\nMINIKUBE LOGS:\n%s\n\nPODS STATUS:\n%s" % \
        (agent_status, agent_container_logs, minikube_logs, pods_status)

def deploy_agent(container, client, configmap_path, daemonset_path, serviceaccount_path, cluster_name="minikube", backend=None, image_name=None, image_tag=None, namespace="default"):
    print("\nPulling %s:%s to the minikube container ..." % (image_name, image_tag))
    pull_agent_image(container, client, image_name=AGENT_IMAGE_NAME, image_tag=AGENT_IMAGE_TAG)
    print("\nDeploying signalfx-agent to the %s cluster ..." % cluster_name)
    serviceaccount_yaml = yaml.load(open(serviceaccount_path).read())
    create_serviceaccount(
        body=serviceaccount_yaml,
        namespace=namespace)
    configmap_yaml = yaml.load(open(configmap_path).read())
    agent_yaml = yaml.load(configmap_yaml['data']['agent.yaml'])
    agent_yaml['globalDimensions']['kubernetes_cluster'] = cluster_name
    agent_yaml['useFullyQualifiedHost'] = False
    if backend:
        agent_yaml['ingestUrl'] = "http://%s:%d" % (get_host_ip(), backend.ingest_port)
        agent_yaml['apiUrl'] = "http://%s:%d" % (get_host_ip(), backend.api_port)
    if 'metricsToExclude' in agent_yaml.keys():
        del agent_yaml['metricsToExclude']
    for monitor in agent_yaml['monitors']:
        if monitor['type'] == 'kubelet-stats':
            monitor['kubeletAPI']['skipVerify'] = True
        if monitor['type'] == 'collectd/etcd':
            monitor['clusterName'] = cluster_name
        if 'metricsToExclude' in monitor.keys():
            del monitor['metricsToExclude']
    configmap_yaml['data']['agent.yaml'] = yaml.dump(agent_yaml)
    create_configmap(
        body=configmap_yaml,
        namespace=namespace)
    daemonset_yaml = yaml.load(open(daemonset_path).read())
    if image_name and image_tag:
        daemonset_yaml['spec']['template']['spec']['containers'][0]['image'] = image_name + ":" + image_tag
    create_daemonset(
        body=daemonset_yaml,
        namespace=namespace)
    assert wait_for(p(has_pod, "signalfx-agent"), timeout_seconds=60), "timed out waiting for the signalfx-agent pod to start!"
    assert wait_for(all_pods_have_ips, timeout_seconds=300), "timed out waiting for pod IPs!"
    agent_container = get_agent_container(client, image_name=AGENT_IMAGE_NAME, image_tag=AGENT_IMAGE_TAG)
    assert agent_container, "failed to get agent container!"
    agent_status = agent_container.status.lower()
    # wait to make sure that the agent container is still running
    time.sleep(10)
    try:
        agent_container.reload()
        agent_status = agent_container.status.lower()
    except:
        agent_status = "exited"
    assert agent_status == 'running', "agent container is not running!\n\n%s\n\n" % (get_all_logs(agent_container, container))

def deploy_services(services=[]):
    for service in services:
        print("\nDeploying %s to the minikube cluster ..." % service)
        exec("%s.deploy()" % service)
    assert wait_for(all_pods_have_ips, timeout_seconds=300), "timed out waiting for pod IPs!"

@contextmanager
@pytest.fixture
def minikube(k8s_version, local_registry, request):
    k8s_timeout = int(request.config.getoption("--k8s-timeout"))
    k8s_container = request.config.getoption("--k8s-container")
    with fake_backend.start(ip=get_host_ip()) as backend:
        if k8s_container:
            print("\nConnecting to %s container ..." % k8s_container)
            container = docker.from_env(version='auto').containers.get(k8s_container)
            assert wait_for(p(container_cmd_exit_0, container, "test -f /kubeconfig"), k8s_timeout), "timed out waiting for minikube to be ready!"
            client = docker.DockerClient(base_url="tcp://%s:2375" % container.attrs["NetworkSettings"]["IPAddress"], version='auto')
            # load kubeconfig from the minikube container
            kube_config.load_kube_config(config_file=get_kubeconfig(container, kubeconfig_path="/kubeconfig"))
            deploy_services(SERVICES)
            deploy_agent(
                container,
                client,
                AGENT_CONFIGMAP_PATH,
                AGENT_DAEMONSET_PATH,
                AGENT_SERVICEACCOUNT_PATH,
                cluster_name="minikube",
                backend=backend,
                image_name=AGENT_IMAGE_NAME,
                image_tag=AGENT_IMAGE_TAG,
                namespace="default")
            yield [container, client, backend]
        else:
            if k8s_version[0] != 'v':
                k8s_version = 'v' + k8s_version
            container_name = "minikube-%s" % k8s_version
            container_options = {
                "name": container_name,
                "privileged": True,
                "environment": {
                    'K8S_VERSION': k8s_version,
                    'TIMEOUT': str(k8s_timeout)
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
            print("\nDeploying minikube %s cluster ..." % k8s_version)
            with run_service('minikube', buildargs={"MINIKUBE_VERSION": MINIKUBE_VERSION}, **container_options) as container:
                #k8s_api_host_port = container.attrs['NetworkSettings']['Ports']['8443/tcp'][0]['HostPort']
                assert wait_for(p(container_cmd_exit_0, container, "test -f /kubeconfig"), k8s_timeout), "timed out waiting for minikube to be ready!"
                client = docker.DockerClient(base_url="tcp://%s:2375" % container.attrs["NetworkSettings"]["IPAddress"], version='auto')
                # load kubeconfig from the minikube container
                kube_config.load_kube_config(config_file=get_kubeconfig(container, kubeconfig_path="/kubeconfig"))
                deploy_services(SERVICES)
                deploy_agent(
                    container,
                    client,
                    AGENT_CONFIGMAP_PATH,
                    AGENT_DAEMONSET_PATH,
                    AGENT_SERVICEACCOUNT_PATH,
                    cluster_name="minikube",
                    backend=backend,
                    image_name=AGENT_IMAGE_NAME,
                    image_tag=AGENT_IMAGE_TAG,
                    namespace="default")
                yield [container, client, backend]


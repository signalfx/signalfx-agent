from functools import partial as p
from contextlib import contextmanager
from kubernetes import config as kube_config

import docker
import os
import pytest
import re
import sys
import yaml

from tests.helpers import fake_backend
from tests.helpers.util import *
from tests.helpers.assertions import *
from tests.kubernetes.utils import *

AGENT_YAMLS_DIR = os.environ.get("AGENT_YAMLS_DIR", "/go/src/github.com/signalfx/signalfx-agent/deployments/k8s")
AGENT_CONFIGMAP_PATH = os.environ.get("AGENT_CONFIGMAP_PATH", os.path.join(AGENT_YAMLS_DIR, "configmap.yaml"))
AGENT_DAEMONSET_PATH = os.environ.get("AGENT_DAEMONSET_PATH", os.path.join(AGENT_YAMLS_DIR, "daemonset.yaml"))
AGENT_SERVICEACCOUNT_PATH = os.environ.get("AGENT_SERVICEACCOUNT_PATH", os.path.join(AGENT_YAMLS_DIR, "serviceaccount.yaml"))
AGENT_IMAGE_NAME = "localhost:5000/signalfx-agent"
AGENT_IMAGE_TAG = "k8s-test"
MINIKUBE_VERSION = os.environ.get("MINIKUBE_VERSION", "latest")

# get metrics to test from docs
DOCS_DIR = os.environ.get("DOCS_DIR", "/go/src/github.com/signalfx/signalfx-agent/docs/monitors")
DOCS = [ 
    "collectd-nginx.md",
    "kubelet-stats.md",
    "kubernetes-cluster.md"
]
DOCS = [os.path.join(DOCS_DIR, i) for i in DOCS]

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

def deploy_nginx(labels={"app": "nginx"}, namespace="default"):
    configmap_data = {"default.conf": '''
        server {
            listen 80;
            server_name  localhost;
            location /nginx_status {
                stub_status on;
                access_log off;
                allow all;
            }
        }'''}
    create_configmap(
        name="nginx-status",
        data=configmap_data,
        labels=labels,
        namespace=namespace)
    pod_template = get_pod_template(
        name="nginx",
        image="nginx:latest",
        port=80,
        labels=labels,
        volume_mounts=[{"name": "nginx-conf", "mount_path": "/etc/nginx/conf.d", "configmap": "nginx-status"}])
    #create_deployment(
    #    name="nginx-deployment",
    #    pod_template=pod_template,
    #    replicas=3,
    #    labels=labels,
    #    namespace=namespace)
    create_replication_controller(
        name="nginx-replication-controller",
        pod_template=pod_template,
        replicas=3,
        labels=labels,
        namespace=namespace)
    create_service(
        name="nginx-service",
        port=80,
        service_type="NodePort",
        labels=labels,
        namespace=namespace)
    assert wait_for(all_pods_have_ips, timeout_seconds=300), "timed out waiting for pod IPs!"

def deploy_agent(configmap_path, daemonset_path, serviceaccount_path, cluster_name="minikube", backend=None, image_name=None, image_tag=None, namespace="default"):
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
                image='registry',
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

@contextmanager
@pytest.fixture
def minikube(k8s_version, request):
    k8s_timeout = int(request.config.getoption("--k8s-timeout"))
    container_name = request.config.getoption("--k8s-container")
    if container_name:
        print("\nConnecting to %s container ..." % container_name)
        container = docker.from_env(version='auto').containers.get(container_name)
        assert wait_for(p(container_cmd_exit_0, container, "test -f /kubeconfig"), k8s_timeout), "timed out waiting for minikube to be ready!"
        client = docker.DockerClient(base_url="tcp://%s:2375" % container.attrs["NetworkSettings"]["IPAddress"], version='auto')
        print("\nPulling %s:%s to the minikube container ..." % (AGENT_IMAGE_NAME, AGENT_IMAGE_TAG))
        pull_agent_image(container, client, image_name=AGENT_IMAGE_NAME, image_tag=AGENT_IMAGE_TAG)
        yield [container, client]
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
        print("\nDeploying minikube ...")
        with run_service('minikube', buildargs={"MINIKUBE_VERSION": MINIKUBE_VERSION}, **container_options) as container:
            #k8s_api_host_port = container.attrs['NetworkSettings']['Ports']['8443/tcp'][0]['HostPort']
            assert wait_for(p(container_cmd_exit_0, container, "test -f /kubeconfig"), k8s_timeout), "timed out waiting for minikube to be ready!"
            client = docker.DockerClient(base_url="tcp://%s:2375" % container.attrs["NetworkSettings"]["IPAddress"], version='auto')
            print("\nPulling %s:%s to the minikube container ..." % (AGENT_IMAGE_NAME, AGENT_IMAGE_TAG))
            pull_agent_image(container, client, image_name=AGENT_IMAGE_NAME, image_tag=AGENT_IMAGE_TAG)
            yield [container, client]

@pytest.mark.k8s
@pytest.mark.kubernetes
def test_k8s_metrics(minikube, local_registry, request):
    metrics_not_found = get_metrics_from_docs(docs=DOCS)
    print("\nCollected %d metrics to test from docs." % len(metrics_not_found))
    metrics_timeout = int(request.config.getoption("--k8s-metrics-timeout"))
    with fake_backend.start(ip=get_host_ip()) as backend:
        with minikube as [mk_container, mk_docker_client]:
            # load kubeconfig from the minikube container
            kube_config.load_kube_config(config_file=get_kubeconfig(mk_container, kubeconfig_path="/kubeconfig"))
            print("\nDeploying nginx to the minikube cluster ...")
            deploy_nginx()
            print("\nDeploying signalfx-agent to the minikube cluster ...")
            deploy_agent(
                AGENT_CONFIGMAP_PATH,
                AGENT_DAEMONSET_PATH,
                AGENT_SERVICEACCOUNT_PATH,
                cluster_name="minikube",
                backend=backend,
                image_name=AGENT_IMAGE_NAME,
                image_tag=AGENT_IMAGE_TAG,
                namespace="default")
            agent_container = get_agent_container(mk_docker_client, image_name=AGENT_IMAGE_NAME, image_tag=AGENT_IMAGE_TAG)
            assert agent_container, "failed to get agent container!"
            agent_status = agent_container.status.lower()
            # wait to make sure that the agent container is still running
            time.sleep(10)
            try:
                agent_container.reload()
                agent_status = agent_container.status.lower()
            except:
                agent_status = "exited"
            assert agent_status == 'running', "agent container is not running!\n\n%s\n\n" % (get_all_logs(agent_container, mk_container))
            # test for metrics
            start_time = time.time()
            while True:
                if len(metrics_not_found) == 0:
                    break
                elif (time.time() - start_time) > metrics_timeout:
                    assert len(metrics_not_found) == 0, "timed out waiting for metric(s) %s!\n\n%s\n\n" % (metrics_not_found, get_all_logs(agent_container, mk_container))
                    break
                else:
                    for metric in metrics_not_found:
                        if has_datapoint_with_metric_name(backend, metric):
                            metrics_not_found.remove(metric)
                            print("Found metric %s" % metric)
                    time.sleep(5)
            # test for dimensions
            nginx_pod = get_all_pods_with_name('nginx-replication-controller-.*')[0]
            nginx_container = mk_docker_client.containers.list(filters={"ancestor": "nginx:latest"})[0]
            expected_dims = [
                {"key": "host", "value": mk_container.attrs['Config']['Hostname']},
                {"key": "container_id", "value": nginx_container.id},
                {"key": "container_name", "value": nginx_container.name},
                {"key": "container_spec_name", "value": nginx_pod.spec.containers[0].name},
                {"key": "kubernetes_namespace", "value": "default"},
                {"key": "kubernetes_cluster", "value": "minikube"},
                {"key": "kubernetes_pod_name", "value": nginx_pod.metadata.name},
                {"key": "kubernetes_pod_uid", "value": nginx_pod.metadata.uid},
                {"key": "machine_id", "value": None},
                {"key": "metric_source", "value": "kubernetes"}
            ]
            for dim in expected_dims:
                if dim["value"]:
                    assert wait_for(p(has_datapoint_with_dim, backend, dim["key"], dim["value"]), timeout_seconds=60), \
                        "timed out waiting for datapoint with dimension %s:%s\n\n%s\n\n" % \
                            (dim["key"], dim["value"], get_all_logs(agent_container, mk_container))
                    print("Found datapoint with dimension %s:%s" % (dim["key"], dim["value"]))
            #print_lines("\n\n%s\n\n" % (get_all_logs(agent_container, mk_container)))


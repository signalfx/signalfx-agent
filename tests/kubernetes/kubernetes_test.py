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

# get metrics to test from docs
DOCS_DIR = os.environ.get("DOCS_DIR", "/go/src/github.com/signalfx/signalfx-agent/docs")
KUBELET_STATS_MD = open(os.path.join(DOCS_DIR, "monitors/kubelet-stats.md")).read()
EXPECTED_KUBELET_STATS_METRICS = re.findall('\| `(.*?)` \| (?:counter|gauge) \|', KUBELET_STATS_MD)
if len(EXPECTED_KUBELET_STATS_METRICS) == 0:
    print("Failed to get metrics from %s!" % os.path.join(DOCS_DIR, "monitors/kubelet-stats.md"))
    sys.exit(1)
KUBERNETES_CLUSTER_MD = open(os.path.join(DOCS_DIR, "monitors/kubernetes-cluster.md")).read()
EXPECTED_KUBERNETES_CLUSTER_METRICS = re.findall('\| `(.*?)` \| (?:counter|gauge) \|', KUBERNETES_CLUSTER_MD)
if len(EXPECTED_KUBERNETES_CLUSTER_METRICS) == 0:
    print("Failed to get metrics from %s!" % os.path.join(DOCS_DIR, "monitors/kubernetes-cluster.md"))
    sys.exit(1)

EXPECTED_DATAPOINTS = [
    {"key": "host", "value": "", "metric": "if_dropped.tx"},
    {"key": "kubernetes_cluster", "value": "minikube", "metric": "memory.free"},
    {"key": "kubernetes_pod_name", "value": "nginx-replication-controller-.*", "metric": "kubernetes.container_ready"},
    {"key": "plugin", "value": "nginx", "metric": "connections.accepted"},
    {"key": "plugin", "value": "nginx", "metric": "connections.handled"},
    {"key": "plugin", "value": "nginx", "metric": "nginx_connections.active"},
    {"key": "plugin", "value": "nginx", "metric": "nginx_connections.reading"},
    {"key": "plugin", "value": "nginx", "metric": "nginx_connections.waiting"},
    {"key": "plugin", "value": "nginx", "metric": "nginx_connections.writing"},
    {"key": "plugin", "value": "nginx", "metric": "nginx_requests"},
]

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
            print("\nAgent image '%s:%s' not found in local registry.\nAttempting to pull from remote registry ..." % \
                (final_agent_image_name, final_agent_image_tag))
            final_image = client.images.pull(final_agent_image_name, tag=final_agent_image_tag)
        except:
            final_image = None
    assert final_image, "agent image '%s:%s' not found!" % (final_agent_image_name, final_agent_image_tag)
    if not env_is_circleci():
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
    if not env_is_circleci():
        print("\nPushing %s:%s ..." % (AGENT_IMAGE_NAME, AGENT_IMAGE_TAG))
        client.images.push(AGENT_IMAGE_NAME, tag=AGENT_IMAGE_TAG)

@contextmanager
@pytest.fixture
def minikube(k8s_version, request):
    k8s_timeout = int(request.config.getoption("--k8s-timeout"))
    container_name = "minikube-%s" % k8s_version
    if env_is_circleci():
        container_options = {
            "name": container_name,
            "privileged": True,
            "environment": {
                'K8S_VERSION': k8s_version,
                'CIRCLECI': 'true',
            },
            "ports": {
                '8443/tcp': None,
                '2375/tcp': None,
            },
            "volumes": {
                "/var/run/docker.sock": {
                    "bind": "/var/run/docker.sock",
                    "mode": "ro"
                },
                "/tmp/scratch": {
                    "bind": "/tmp/scratch",
                    "mode": "rw"
                },
            }
        }
    else:
        container_options = {
            "name": container_name,
            "privileged": True,
            "environment": {
                'K8S_VERSION': k8s_version,
                'CIRCLECI': 'false',
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
    with run_service('minikube', **container_options) as mk:
        #k8s_api_host_port = mk.attrs['NetworkSettings']['Ports']['8443/tcp'][0]['HostPort']
        assert wait_for(p(container_cmd_exit_0, mk, "test -f /kubeconfig"), k8s_timeout), "timed out waiting for minikube to be ready!"
        if env_is_circleci():
            client = docker.from_env(version='auto')
        else:
            client = docker.DockerClient(base_url="tcp://%s:2375" % mk.attrs["NetworkSettings"]["IPAddress"], version='auto')
            print("\nPulling %s:%s to the minikube container ..." % (AGENT_IMAGE_NAME, AGENT_IMAGE_TAG))
            pull_agent_image(mk, client, image_name=AGENT_IMAGE_NAME, image_tag=AGENT_IMAGE_TAG)
        yield [mk, client]

@pytest.mark.k8s
@pytest.mark.kubernetes
def test_k8s_metrics(minikube, local_registry, request):
    metrics_timeout = int(request.config.getoption("--k8s-metrics-timeout"))
    with fake_backend.start(ip=get_host_ip()) as backend:
        with minikube as [mk, mk_docker_client]:
            kube_config.load_kube_config(config_file=get_kubeconfig(mk, kubeconfig_path="/kubeconfig"))
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
            assert agent_status == 'running', "agent container is not running!\n\n%s\n\n" % (get_all_logs(agent_container, mk))
            # test for metrics
            metrics_not_found = EXPECTED_KUBELET_STATS_METRICS + EXPECTED_KUBERNETES_CLUSTER_METRICS
            start_time = time.time()
            while True:
                if len(metrics_not_found) == 0:
                    break
                elif (time.time() - start_time) > metrics_timeout:
                    assert len(metrics_not_found) == 0, "timed out waiting for metric(s) %s!\n\n%s\n\n" % (metrics_not_found, get_all_logs(agent_container, mk))
                    break
                else:
                    for metric in metrics_not_found:
                        if has_datapoint_with_metric_name(backend, metric):
                            metrics_not_found.remove(metric)
                            print("Found metric %s" % metric)
                    time.sleep(5)
            # test for datapoints
            for dp in EXPECTED_DATAPOINTS:
                if dp["key"] == "host":
                    dp["value"] = mk.attrs['Config']['Hostname']
                assert wait_for(p(has_datapoint_with_dim_and_metric_name, backend, dp["key"], dp["value"], dp["metric"]), timeout_seconds=120), \
                    "timed out waiting for datapoint %s:%s:%s\n\n%s\n\n" % (dp["key"], dp["value"], dp["metric"], get_all_logs(agent_container, mk))
                print("Found datapoint %s:%s:%s" % (dp["key"], dp["value"], dp["metric"]))
            print_lines("\n\n%s\n\n" % (get_all_logs(agent_container, mk)))


from kubernetes import config as kube_config
from tests.helpers.util import *
from tests.kubernetes.utils import *

import docker
import pytest

@pytest.fixture(scope="module")
def backend():
    with fake_backend.start(ip=get_host_ip()) as backend:
        print("\nStarting fake backend ...")
        yield backend

@pytest.fixture(scope="module")
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
    yield
    try:
        client.containers.get("registry").remove(force=True)
    except:
        pass

@pytest.fixture(scope="module")
def minikube(request, backend, local_registry):
    k8s_version = request.param
    k8s_timeout = int(request.config.getoption("--k8s-timeout"))
    k8s_container = request.config.getoption("--k8s-container")
    container = get_minikube_container(k8s_version, k8s_timeout, k8s_container)
    client = get_minikube_docker_client(container)
    # load kubeconfig from the minikube container
    kube_config.load_kube_config(config_file=get_kubeconfig(container, kubeconfig_path="/kubeconfig"))
    if len(get_all_pods_with_name('^signalfx-agent.*')) == 0:
        deploy_services(SERVICES)
        agent_container = deploy_agent(
            container,
            AGENT_CONFIGMAP_PATH,
            AGENT_DAEMONSET_PATH,
            AGENT_SERVICEACCOUNT_PATH,
            cluster_name="minikube",
            backend=backend,
            image_name=AGENT_IMAGE_NAME,
            image_tag=AGENT_IMAGE_TAG,
            namespace="default")
    def teardown():
        if not k8s_container:
            try:
                container.remove(force=True, v=True)
            except:
                pass
    request.addfinalizer(teardown)
    return container


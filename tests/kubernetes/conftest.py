from tests.kubernetes.minikube import *
from tests.kubernetes.utils import *
import docker
import pytest


@pytest.fixture(scope="module")
def local_registry():
    client = get_docker_client()
    cont = None
    try:
        client.containers.get("registry")
        print("\nRegistry container localhost:5000 already running")
    except:
        print("\nStarting registry container localhost:5000")
        cont = client.containers.run(
            image='registry:latest',
            name='registry',
            detach=True,
            ports={'5000/tcp': 5000})
        assert wait_for(lambda: has_log_message(cont.logs().decode('utf-8'), message="listening on [::]:5000"), timeout_seconds=5), \
            "timed out waiting for registry container to be ready!\n\n%s\n" % cont.logs().decode('utf-8')
    try:
        yield
    finally:
        if cont:
            cont.remove(force=True)


@pytest.fixture(scope="module")
def agent_image(local_registry, request):
    client = get_docker_client()
    final_agent_image_name = request.config.getoption("--k8s-agent-name")
    final_agent_image_tag = request.config.getoption("--k8s-agent-tag")
    agent_image_name = "localhost:5000/%s" % final_agent_image_name.split("/")[-1]
    agent_image_tag = final_agent_image_tag
    try:
        final_agent_image = client.images.get(final_agent_image_name + ":" + final_agent_image_tag)
    except:
        try:
            print("\nAgent image '%s:%s' not found in local registry." % (final_agent_image_name, final_agent_image_tag))
            print("\nAttempting to pull from remote registry ...")
            final_agent_image = client.images.pull(final_agent_image_name, tag=final_agent_image_tag)
        except:
            final_agent_image = None
    assert final_agent_image, "agent image '%s:%s' not found!" % (final_agent_image_name, final_agent_image_tag)
    print("\nTagging %s:%s as %s:%s ..." % (final_agent_image_name, final_agent_image_tag, agent_image_name, agent_image_tag))
    final_agent_image.tag(agent_image_name, tag=agent_image_tag)
    print("\nPushing %s:%s ..." % (agent_image_name, agent_image_tag))
    client.images.push(agent_image_name, tag=agent_image_tag)
    return {"name": agent_image_name, "tag": agent_image_tag}


@pytest.fixture(scope="module")
def minikube(request):
    k8s_version = request.param
    k8s_timeout = int(request.config.getoption("--k8s-timeout"))
    k8s_container = request.config.getoption("--k8s-container")
    k8s_skip_teardown = request.config.getoption("--k8s-skip-teardown")
    mk = Minikube()

    def teardown():
        if not k8s_container and not k8s_skip_teardown:
            try:
                mk.container.remove(force=True, v=True)
            except:
                pass

    request.addfinalizer(teardown)
    if k8s_container:
        mk.connect(k8s_container, k8s_timeout)
    else:
        mk.deploy(k8s_version, k8s_timeout)
    return mk


@pytest.fixture
def k8s_observer(request):
    return request.param


@pytest.fixture
def k8s_test_timeout(request):
    return int(request.config.getoption("--k8s-test-timeout"))


@pytest.fixture
def k8s_monitor_without_endpoints(request):
    try:
        return request.param
    except:
        pytest.skip("no monitors to test")
        return None


@pytest.fixture
def k8s_monitor_with_endpoints(request):
    try:
        return request.param
    except:
        pytest.skip("no monitors to test")
        return None

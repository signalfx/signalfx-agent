from tests.kubernetes.minikube import *
from tests.kubernetes.utils import *
import docker
import pytest


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
    agent_image_name = getattr(request.module, "AGENT_IMAGE_NAME")
    agent_image_tag = getattr(request.module, "AGENT_IMAGE_TAG")
    print("\nTagging %s:%s as %s:%s ..." % (final_agent_image_name, final_agent_image_tag, agent_image_name, agent_image_tag))
    final_image.tag(agent_image_name, tag=agent_image_tag)
    print("\nPushing %s:%s ..." % (agent_image_name, agent_image_tag))
    client.images.push(agent_image_name, tag=agent_image_tag)
    yield
    try:
        client.containers.get("registry").remove(force=True)
    except:
        pass


@pytest.fixture(scope="module")
def minikube(local_registry, request):
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
    try:
        mk.create_secret("signalfx-agent", "testing123")
    except:
        pass
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

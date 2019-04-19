import os
import random
import string
import subprocess
import time
from functools import partial as p

import pytest

from tests.helpers.kubernetes.minikube import Minikube, has_docker_image
from tests.helpers.util import wait_for, get_docker_client, run_container

REPO_ROOT_DIR = os.path.join(os.path.dirname(os.path.realpath(__file__)), "..")
K8S_DEFAULT_VERSION = "1.14.0"
K8S_DEFAULT_TIMEOUT = int(os.environ.get("K8S_TIMEOUT", 300))
K8S_DEFAULT_TEST_TIMEOUT = 60
K8S_SUPPORTED_OBSERVERS = ["k8s-api", "k8s-kubelet"]
K8S_DEFAULT_OBSERVERS = K8S_SUPPORTED_OBSERVERS
K8S_SFX_AGENT_BUILD_TIMEOUT = int(os.environ.get("K8S_SFX_AGENT_BUILD_TIMEOUT", 600))

pytest.register_assert_rewrite("tests.helpers.verify")


# pylint: disable=line-too-long
def pytest_addoption(parser):
    parser.addoption(
        "--k8s-version",
        action="store",
        default=K8S_DEFAULT_VERSION,
        help="K8S cluster version for minikube to deploy (default=%s). This option is ignored if the --k8s-container option also is specified."
        % K8S_DEFAULT_VERSION,
    )
    parser.addoption(
        "--k8s-observers",
        action="store",
        default=",".join(K8S_DEFAULT_OBSERVERS),
        help="Comma-separated string of observers to test monitors with endpoints for the SignalFx agent (default=%s). Use '--k8s-observers=all' to test all supported observers."
        % ",".join(K8S_DEFAULT_OBSERVERS),
    )
    parser.addoption(
        "--k8s-timeout",
        action="store",
        default=K8S_DEFAULT_TIMEOUT,
        help="Timeout (in seconds) to wait for the minikube cluster to be ready (default=%d)." % K8S_DEFAULT_TIMEOUT,
    )
    parser.addoption(
        "--k8s-sfx-agent",
        action="store",
        default=None,
        help="SignalFx agent image name:tag to use for K8S tests. If not specified, the agent image will be built from the local source code.",
    )
    parser.addoption(
        "--k8s-test-timeout",
        action="store",
        default=K8S_DEFAULT_TEST_TIMEOUT,
        help="Timeout (in seconds) for each K8S test (default=%d)." % K8S_DEFAULT_TEST_TIMEOUT,
    )
    parser.addoption(
        "--k8s-container",
        action="store",
        default=None,
        help="Name of a running minikube container to use for the tests (the container should not have the agent or any services already running). If not specified, a new minikube container will automatically be deployed.",
    )
    parser.addoption(
        "--k8s-skip-teardown",
        action="store_true",
        help="If specified, the minikube container will not be stopped/removed when the tests complete.",
    )


def pytest_generate_tests(metafunc):
    if "k8s_observer" in metafunc.fixturenames:
        k8s_observers = metafunc.config.getoption("--k8s-observers")
        if not k8s_observers:
            observers_to_test = K8S_DEFAULT_OBSERVERS
        elif k8s_observers.lower() == "all":
            observers_to_test = K8S_SUPPORTED_OBSERVERS
        else:
            for obs in k8s_observers.split(","):
                assert obs in K8S_SUPPORTED_OBSERVERS, 'observer "%s" not supported!' % obs
            observers_to_test = k8s_observers.split(",")
        metafunc.parametrize("k8s_observer", observers_to_test, indirect=True)


@pytest.fixture(scope="session")
def minikube(request, worker_id):
    def teardown():
        if inst.container and not k8s_skip_teardown:
            print("Removing %s container ..." % inst.container.name)
            inst.container.remove(force=True, v=True)

    request.addfinalizer(teardown)
    k8s_version = os.environ.get("K8S_VERSION")
    if not k8s_version:
        k8s_version = request.config.getoption("--k8s-version")
    k8s_timeout = request.config.getoption("--k8s-timeout")
    if not k8s_timeout:
        k8s_timeout = K8S_DEFAULT_TIMEOUT
    else:
        k8s_timeout = int(k8s_timeout)
    k8s_container = request.config.getoption("--k8s-container")
    k8s_skip_teardown = request.config.getoption("--k8s-skip-teardown")
    inst = Minikube()
    if k8s_container:
        # connect to existing minikube container and cluster
        k8s_skip_teardown = True
        inst.connect(name=k8s_container, timeout=k8s_timeout)
    elif worker_id in ("master", "gw0"):
        # deploy new minikube container and cluster
        if int(os.environ.get("PYTEST_XDIST_WORKER_COUNT", 1)) > 1:
            k8s_skip_teardown = True
        inst.deploy(k8s_version, timeout=k8s_timeout)
    else:
        # connect to minikube container and cluster deployed by gw0 worker
        time.sleep(10)  # wait for gw0 to clean and initialize the environment
        k8s_skip_teardown = True
        inst.connect(k8s_version=k8s_version, timeout=k8s_timeout)
    return inst


@pytest.fixture(scope="session")
def agent_image(minikube, request, worker_id):  # pylint: disable=redefined-outer-name
    def teardown():
        if temp_agent_name and temp_agent_tag:
            try:
                client.images.remove("%s:%s" % (temp_agent_name, temp_agent_tag))
            except:  # noqa pylint: disable=bare-except
                pass

    request.addfinalizer(teardown)
    sfx_agent_name = os.environ.get("K8S_SFX_AGENT")
    if not sfx_agent_name:
        sfx_agent_name = request.config.getoption("--k8s-sfx-agent")
    if sfx_agent_name:
        try:
            agent_image_name, agent_image_tag = sfx_agent_name.rsplit(":", maxsplit=1)
        except ValueError:
            agent_image_name = sfx_agent_name
            agent_image_tag = "latest"
    else:
        agent_image_name = "signalfx-agent"
        agent_image_tag = "k8s-test"
    temp_agent_name = None
    temp_agent_tag = None
    client = get_docker_client()
    if worker_id in ("master", "gw0"):
        if sfx_agent_name and not has_docker_image(client, sfx_agent_name):
            print('\nAgent image "%s" not found in local registry.' % sfx_agent_name)
            print("Attempting to pull from remote registry to minikube ...")
            sfx_agent_image = minikube.pull_agent_image(agent_image_name, agent_image_tag)
            _, output = minikube.container.exec_run("docker images")
            print(output.decode("utf-8"))
            return {"name": agent_image_name, "tag": agent_image_tag, "id": sfx_agent_image.id}

        if sfx_agent_name:
            print('\nAgent image "%s" found in local registry.' % sfx_agent_name)
            sfx_agent_image = client.images.get(sfx_agent_name)
        else:
            print(
                '\nBuilding agent image from local source and tagging as "%s:%s" ...'
                % (agent_image_name, agent_image_tag)
            )
            subprocess.run(
                "make image",
                shell=True,
                env={"PULL_CACHE": "yes", "AGENT_IMAGE_NAME": agent_image_name, "AGENT_VERSION": agent_image_tag},
                stderr=subprocess.STDOUT,
                check=True,
                cwd=REPO_ROOT_DIR,
                timeout=K8S_SFX_AGENT_BUILD_TIMEOUT,
            )
            sfx_agent_image = client.images.get(agent_image_name + ":" + agent_image_tag)
        temp_agent_name = "localhost:%d/signalfx-agent-dev" % minikube.registry_port
        temp_agent_tag = "latest"
        print("\nPushing agent image to minikube ...")
        sfx_agent_image.tag(temp_agent_name, tag=temp_agent_tag)
        client.images.push(temp_agent_name, tag=temp_agent_tag)
        sfx_agent_image = minikube.pull_agent_image(temp_agent_name, temp_agent_tag, sfx_agent_image.id)
        sfx_agent_image.tag(agent_image_name, tag=agent_image_tag)
        _, output = minikube.container.exec_run("docker images")
        print(output.decode("utf-8").strip())
    else:
        print("\nWaiting for agent image to be built/pulled to minikube ...")
        assert wait_for(
            p(has_docker_image, minikube.client, agent_image_name, agent_image_tag),
            timeout_seconds=K8S_SFX_AGENT_BUILD_TIMEOUT,
            interval_seconds=2,
        ), 'timed out waiting for agent image "%s:%s"!' % (agent_image_name, agent_image_tag)
        sfx_agent_image = minikube.client.images.get(agent_image_name + ":" + agent_image_tag)
    return {"name": agent_image_name, "tag": agent_image_tag, "id": sfx_agent_image.id}


@pytest.fixture
def k8s_observer(request):
    return request.param


@pytest.fixture
def k8s_test_timeout(request):
    return int(request.config.getoption("--k8s-test-timeout"))


@pytest.fixture
def k8s_namespace(worker_id):
    chars = string.ascii_lowercase + string.digits
    namespace = worker_id + "-" + "".join((random.choice(chars)) for x in range(8))
    return namespace


@pytest.fixture
def devstack():
    devstack_opts = dict(
        entrypoint="/lib/systemd/systemd",
        privileged=True,
        volumes={
            "/lib/modules": {"bind": "/lib/modules", "mode": "ro"},
            "/sys/fs/cgroup": {"bind": "/sys/fs/cgroup", "mode": "ro"},
        },
        environment={"container": "docker"},
    )
    with run_container("quay.io/signalfx/devstack:latest", **devstack_opts) as container:
        code, output = container.exec_run("start-devstack.sh")
        assert code == 0, "devstack failed to start:\n%s" % output.decode("utf-8")
        yield container

import random
import re
import string
import subprocess
import time
import urllib
from functools import partial as p

import pytest

from helpers.assertions import has_log_message
from helpers.kubernetes.minikube import Minikube
from helpers.kubernetes.utils import container_is_running, has_docker_image
from helpers.util import wait_for, get_docker_client, run_container


def get_latest_k8s_version(url="https://storage.googleapis.com/kubernetes-release/release/stable.txt", max_attempts=3):
    version = None
    attempt = 0
    while attempt < max_attempts:
        attempt += 1
        try:
            with urllib.request.urlopen(url) as resp:
                version = resp.read().decode("utf-8").strip()
            break
        except urllib.error.HTTPError as e:
            if attempt == max_attempts:
                raise e
        time.sleep(5)
        version = None
    assert version, "Failed to get latest K8S version from %s" % url
    assert re.match(r"^v?\d+\.\d+\.\d+$", version), "Unknown version '%s' from %s" % (version, url)
    return version


K8S_MIN_VERSION = "1.7.0"
K8S_LATEST_VERSION = get_latest_k8s_version()
K8S_DEFAULT_VERSION = re.sub(r"\.\d+$", ".0", K8S_LATEST_VERSION)
K8S_DEFAULT_TIMEOUT = 300
K8S_DEFAULT_TEST_TIMEOUT = 120
K8S_DEFAULT_CONTAINER_NAME = "minikube"
K8S_SUPPORTED_OBSERVERS = ["k8s-api", "k8s-kubelet"]
K8S_DEFAULT_OBSERVERS = K8S_SUPPORTED_OBSERVERS


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
    if "minikube" in metafunc.fixturenames:
        k8s_version = metafunc.config.getoption("--k8s-version")
        if not k8s_version:
            version_to_test = K8S_DEFAULT_VERSION
        elif k8s_version.lower() == "latest":
            version_to_test = K8S_LATEST_VERSION
        else:
            version_to_test = k8s_version
        assert re.match(r"^v?\d+\.\d+\.\d+$", version_to_test), "Invalid K8S version '%s'" % version_to_test
        metafunc.parametrize(
            "minikube",
            [version_to_test],
            ids=["v%s" % v.strip("v") for v in [version_to_test]],
            scope="session",
            indirect=True,
        )
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
        metafunc.parametrize("k8s_observer", observers_to_test, ids=[o for o in observers_to_test], indirect=True)


@pytest.fixture(scope="session")
def minikube(request, worker_id):
    def teardown():
        if not k8s_skip_teardown:
            try:
                print("Removing minikube container ...")
                inst.container.remove(force=True, v=True)
            except:  # noqa pylint: disable=bare-except
                pass

    request.addfinalizer(teardown)
    k8s_version = "v" + request.param.lstrip("v")
    k8s_timeout = int(request.config.getoption("--k8s-timeout"))
    k8s_container = request.config.getoption("--k8s-container")
    k8s_skip_teardown = request.config.getoption("--k8s-skip-teardown")
    inst = Minikube()
    inst.worker_id = worker_id
    if k8s_container:
        inst.connect(k8s_container, k8s_timeout)
        k8s_skip_teardown = True
    elif worker_id in ("master", "gw0"):
        inst.deploy(k8s_version, k8s_timeout)
        if worker_id == "gw0":
            k8s_skip_teardown = True
    else:
        inst.connect(K8S_DEFAULT_CONTAINER_NAME, k8s_timeout, version=k8s_version)
        k8s_skip_teardown = True
    return inst


@pytest.fixture(scope="session")
def registry(minikube, worker_id, request):  # pylint: disable=redefined-outer-name
    def get_registry_logs():
        cont.reload()
        return cont.logs().decode("utf-8")

    cont = None
    port = None
    k8s_container = request.config.getoption("--k8s-container")
    if not k8s_container and worker_id in ("master", "gw0"):
        minikube.start_registry()
        port = minikube.registry_port
    print("\nWaiting for registry to be ready ...")
    assert wait_for(
        p(container_is_running, minikube.client, "registry"), timeout_seconds=60, interval_seconds=2
    ), "timed out waiting for registry container to start!"
    cont = minikube.client.containers.get("registry")
    assert wait_for(
        lambda: has_log_message(get_registry_logs(), message="listening on [::]:"),
        timeout_seconds=30,
        interval_seconds=2,
    ), "timed out waiting for registry to be ready!"
    if not port:
        match = re.search(r"listening on \[::\]:(\d+)", get_registry_logs())
        assert match, "failed to determine registry port!"
        port = int(match.group(1))
    return {"container": cont, "port": port}


@pytest.fixture(scope="session")
def agent_image(minikube, registry, request, worker_id):  # pylint: disable=redefined-outer-name
    def teardown():
        if temp_agent_name and temp_agent_tag:
            try:
                client.images.remove("%s:%s" % (temp_agent_name, temp_agent_tag))
            except:  # noqa pylint: disable=bare-except
                pass

    request.addfinalizer(teardown)
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
            )
            sfx_agent_image = client.images.get(agent_image_name + ":" + agent_image_tag)
        temp_agent_name = "localhost:%d/signalfx-agent-dev" % registry["port"]
        temp_agent_tag = "latest"
        print("\nPushing agent image to minikube ...")
        sfx_agent_image.tag(temp_agent_name, tag=temp_agent_tag)
        client.images.push(temp_agent_name, tag=temp_agent_tag)
        sfx_agent_image = minikube.pull_agent_image(temp_agent_name, temp_agent_tag, sfx_agent_image.id)
        sfx_agent_image.tag(agent_image_name, tag=agent_image_tag)
        _, output = minikube.container.exec_run("docker images")
        print(output.decode("utf-8"))
    else:
        print("\nWaiting for agent image to be built/pulled to minikube ...")
        assert wait_for(
            p(has_docker_image, minikube.client, agent_image_name, agent_image_tag),
            timeout_seconds=600,
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

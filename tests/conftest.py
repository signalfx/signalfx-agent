import os
import tempfile

import pytest
from tests.helpers.kubernetes.cluster import Cluster
from tests.helpers.util import get_docker_client, run_container

MINIKUBE_DEFAULT_TIMEOUT = int(os.environ.get("MINIKUBE_TIMEOUT", 300))
K8S_DEFAULT_TEST_TIMEOUT = 60
NON_INTEGRATION_MARKERS = {"packaging", "installer", "kubernetes", "windows_only", "deployment", "perf_test", "bundle"}


def pytest_collection_modifyitems(items):
    for item in items:
        if isinstance(item, pytest.Function):
            if "k8s_cluster" in item.fixturenames:
                item.add_marker("kubernetes")
            markers = {marker.name for marker in item.iter_markers()}
            if not NON_INTEGRATION_MARKERS.intersection(markers):
                item.add_marker("integration")


pytest.register_assert_rewrite("tests.helpers.verify")

# pylint: disable=line-too-long
def pytest_addoption(parser):
    parser.addoption(
        "--agent-image-name",
        action="store",
        default="",
        help="SignalFx agent image name:tag to use for K8s tests. Defaults to the agent image built by the `make push-minikube-agent` command if using minikube. "
        "Must be specified if --use-minikube is false.",
    )
    parser.addoption(
        "--kubeconfig",
        action="store",
        default=None,
        help="Equivalent to the KUBECONFIG envvar used by kubectl.  Only relevant if --use-minikube=false.",
    )
    parser.addoption(
        "--kube-context",
        action="store",
        default=None,
        help="The kubeconfig context to use, only relevant if --use-minikube=false.  Defaults to the current "
        "context in the configured kube config file.",
    )
    parser.addoption(
        "--minikube-container-name",
        action="store",
        default="minikube",
        help="Name of a running minikube container to use for the tests (the container should not have the agent or any services already running).",
    )
    parser.addoption(
        "--no-use-minikube",
        action="store_true",
        default=False,
        help="If provided, the Kubernetes cluster used for the tests must be provided by specifying "
        "the --kubeconfig (and optionally the --kube-context) option.  Otherwise, a local minikube "
        "container will be used to run the tests. "
        "If not provided, you must start minikube before running this test suite by running `make run-minikube`.",
    )
    parser.addoption(
        "--test-bundle-path",
        action="store",
        help="Path to a bundle .tar.gz file for testing.  Required for tests that need a bundle.",
    )


@pytest.fixture
def k8s_cluster(request):
    agent_image_name = request.config.getoption("--agent-image-name")

    kube_context = None
    if request.config.getoption("--no-use-minikube"):
        assert agent_image_name, "You must specify the agent image name when not using minikube"
        kube_config_path = request.config.getoption("--kubeconfig")
        kube_context = request.config.getoption("--kube-context")
    else:
        minikube_container_name = request.config.getoption("--minikube-container-name")
        dclient = get_docker_client()
        assert get_docker_client().containers.get(
            minikube_container_name
        ), f"You must start the minikube container ({minikube_container_name}) before running pytest by running `make minikube`."

        # Pull the kubeconfig out of the minikube container and make a cluster instance based on it.
        minikube_cont = dclient.containers.get(minikube_container_name)
        _, kubeconfig_bytes = minikube_cont.exec_run("cat /kubeconfig")

        _, kube_config_path = tempfile.mkstemp()
        request.addfinalizer(lambda: os.remove(kube_config_path))

        with open(kube_config_path, "wb") as fd:
            fd.write(kubeconfig_bytes)
        if not agent_image_name:
            agent_image_name = "localhost:5000/signalfx-agent:latest"

    cluster = Cluster(kube_config_path=kube_config_path, kube_context=kube_context, agent_image_name=agent_image_name)
    request.addfinalizer(cluster.delete_test_namespace)
    return cluster


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

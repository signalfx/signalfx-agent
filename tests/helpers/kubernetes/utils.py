import os
import re
import socket
from contextlib import contextmanager
from functools import partial as p

import docker
import yaml
from kubernetes import client as kube_client
from kubernetes.client.rest import ApiException

from helpers.assertions import has_any_metric_or_dim
from helpers.formatting import print_dp_or_event
from helpers.util import container_ip, fake_backend, get_host_ip, get_observer_dims_from_selfdescribe, wait_for

CUR_DIR = os.path.dirname(os.path.realpath(__file__))
AGENT_YAMLS_DIR = os.environ.get("AGENT_YAMLS_DIR", os.path.realpath(os.path.join(CUR_DIR, "../../../deployments/k8s")))
AGENT_CONFIGMAP_PATH = os.environ.get("AGENT_CONFIGMAP_PATH", os.path.join(AGENT_YAMLS_DIR, "configmap.yaml"))
AGENT_DAEMONSET_PATH = os.environ.get("AGENT_DAEMONSET_PATH", os.path.join(AGENT_YAMLS_DIR, "daemonset.yaml"))
AGENT_SERVICEACCOUNT_PATH = os.environ.get(
    "AGENT_SERVICEACCOUNT_PATH", os.path.join(AGENT_YAMLS_DIR, "serviceaccount.yaml")
)
K8S_CREATE_TIMEOUT = 180
K8S_DELETE_TIMEOUT = 10


def run_k8s_monitors_test(  # pylint: disable=too-many-locals,too-many-arguments,dangerous-default-value
    agent_image,
    minikube,
    monitors,
    observer=None,
    namespace="default",
    yamls=None,
    yamls_timeout=K8S_CREATE_TIMEOUT,
    expected_metrics=set(),
    expected_dims=set(),
    passwords=["testing123"],
    test_timeout=60,
):
    """
    Wrapper function for K8S setup and tests within minikube for monitors.

    Setup includes starting the fake backend, creating K8S deployments, and smart agent configuration/deployment
    within minikube.

    Tests include waiting for at least one metric and/or dimension from the expected_metrics and expected_dims args,
    and checking for cleartext passwords in the output from the agent status and agent container logs.

    Args:
    agent_image (dict):                    Dict object from the agent_image fixture
    minikube (Minikube):                   Minkube object from the minikube fixture
    monitors (str, dict, or list of dict): YAML-based definition of monitor(s) for the smart agent agent.yaml
    observer (str):                        Observer for the smart agent agent.yaml (if None,
                                             the agent.yaml will not be configured for an observer)
    namespace (str):                       K8S namespace for the smart agent and deployments
    yamls (list of str):                   Path(s) to K8S deployment yamls to create
    yamls_timeout (int):                   Timeout in seconds to wait for the K8S deployments to be ready
    expected_metrics (set of str):         Set of metric names to test for (if empty or None,
                                             metrics test will be skipped)
    expected_dims (set of str):            Set of dimension keys to test for (if None, dimensions test will be skipped)
    passwords (list of str):               Cleartext password(s) to test for in the output from the agent status and
                                             agent container logs
    test_timeout (int):                    Timeout in seconds to wait for metrics/dimensions
    """
    if yamls is None:
        yamls = []
    if expected_dims is not None:
        if observer:
            expected_dims = expected_dims.union(get_observer_dims_from_selfdescribe(observer))
        expected_dims = expected_dims.union({"kubernetes_cluster"})

    with run_k8s_with_agent(agent_image, minikube, monitors, observer, namespace, yamls, yamls_timeout) as [
        backend,
        agent,
    ]:
        assert wait_for(
            p(has_any_metric_or_dim, backend, expected_metrics, expected_dims), timeout_seconds=test_timeout
        ), (
            "timed out waiting for metrics in %s with any dimensions in %s!\n\n"
            "AGENT STATUS:\n%s\n\n"
            "AGENT CONTAINER LOGS:\n%s\n"
            % (expected_metrics, expected_dims, agent.get_status(), agent.get_container_logs())
        )
        agent_status = agent.get_status()
        container_logs = agent.get_container_logs()
        assert all([p not in agent_status for p in passwords]), (
            "cleartext password(s) found in agent-status output!\n\n%s\n" % agent_status
        )
        assert all([p not in container_logs for p in passwords]), (
            "cleartext password(s) found in agent container logs!\n\n%s\n" % container_logs
        )


@contextmanager
def run_k8s_with_agent(
    agent_image, minikube, monitors, observer=None, namespace="default", yamls=None, yamls_timeout=K8S_CREATE_TIMEOUT
):
    """
    Runs a minikube environment with the agent and a set of specified
    resources.
    """
    if yamls is None:
        yamls = []
    try:
        monitors = yaml.load(monitors)
    except AttributeError:
        pass
    if isinstance(monitors, dict):
        monitors = [monitors]
    assert isinstance(monitors, list), "unknown type/defintion for monitors:\n%s\n" % monitors
    with fake_backend.start(ip_addr=get_host_ip()) as backend:
        with minikube.deploy_k8s_yamls(yamls, namespace=namespace, timeout=yamls_timeout):
            with minikube.deploy_agent(
                AGENT_CONFIGMAP_PATH,
                AGENT_DAEMONSET_PATH,
                AGENT_SERVICEACCOUNT_PATH,
                observer=observer,
                monitors=monitors,
                cluster_name="minikube",
                backend=backend,
                image_name=agent_image["name"],
                image_tag=agent_image["tag"],
                namespace=namespace,
            ) as agent:
                try:
                    yield [backend, agent]
                finally:
                    print("\nDatapoints received:")
                    for dp in backend.datapoints:
                        print_dp_or_event(dp)
                    print("\nEvents received:")
                    for event in backend.events:
                        print_dp_or_event(event)


def has_namespace(name):
    api = kube_client.CoreV1Api()
    try:
        api.read_namespace(name=name)
        return True
    except ApiException as e:
        if e.status == 404:
            return False
        raise


def create_namespace(name):
    api = kube_client.CoreV1Api()
    return api.create_namespace(body=kube_client.V1Namespace(metadata=kube_client.V1ObjectMeta(name=name)))


def has_secret(name, namespace="default"):
    api = kube_client.CoreV1Api()
    try:
        api.read_namespaced_secret(name, namespace=namespace)
        return True
    except ApiException as e:
        if e.status == 404:
            return False
        raise


def create_secret(name, key, value, namespace="default"):
    if not has_namespace(namespace):
        create_namespace(namespace)
    api = kube_client.CoreV1Api()
    return api.create_namespaced_secret(
        body=kube_client.V1Secret(
            metadata=kube_client.V1ObjectMeta(name=name, namespace=namespace), string_data={key: value}
        ),
        namespace=namespace,
    )


def has_serviceaccount(name, namespace="default"):
    api = kube_client.CoreV1Api()
    try:
        api.read_namespaced_service_account(name, namespace=namespace)
        return True
    except ApiException as e:
        if e.status == 404:
            return False
        raise


def create_serviceaccount(body, namespace=None, timeout=K8S_CREATE_TIMEOUT):
    api = kube_client.CoreV1Api()
    name = body["metadata"]["name"]
    body["apiVersion"] = "v1"
    if namespace:
        body["metadata"]["namespace"] = namespace
    else:
        try:
            namespace = body["metadata"]["namespace"]
        except KeyError:
            namespace = "default"
    if not has_namespace(namespace):
        create_namespace(namespace)
    serviceaccount = api.create_namespaced_service_account(body=body, namespace=namespace)
    assert wait_for(p(has_serviceaccount, name, namespace=namespace), timeout_seconds=timeout), (
        'timed out waiting for service account "%s" to be created!' % name
    )
    return serviceaccount


def api_client_from_version(api_version):
    return {"v1": kube_client.CoreV1Api(), "extensions/v1beta1": kube_client.ExtensionsV1beta1Api()}[api_version]


def camel_case_to_snake_case(name):
    """
    Useful for converting K8s "kind" field values to the k8s api method name
    """
    return re.sub("([a-z0-9])([A-Z])", r"\1_\2", re.sub("(.)([A-Z][a-z]+)", r"\1_\2", name)).lower()


def has_resource(name, kind, api_client, namespace="default"):
    """
    Returns True if the resource exists.  `kind` should be the same thing that
    goes in the `kind` field of the k8s resource.
    """
    try:
        getattr(api_client, "read_namespaced_" + camel_case_to_snake_case(kind))(name, namespace=namespace)
        return True
    except ApiException as e:
        if e.status == 404:
            return False
        raise


def create_resource(body, api_client, namespace=None, timeout=K8S_CREATE_TIMEOUT):
    name = body["metadata"]["name"]
    kind = body["kind"]
    # The namespace in the resource body always takes precidence
    namespace = body.get("metadata", {}).get("namespace", namespace)

    if not has_namespace(namespace):
        create_namespace(namespace)
    resource = getattr(api_client, "create_namespaced_" + camel_case_to_snake_case(kind))(
        body=body, namespace=namespace
    )
    assert wait_for(
        p(has_resource, name, kind, api_client, namespace=namespace), timeout_seconds=timeout
    ), 'timed out waiting for %s "%s" to be created!' % (kind, name)
    return resource


def patch_resource(body, api_client, namespace=None, timeout=K8S_CREATE_TIMEOUT):
    name = body["metadata"]["name"]
    kind = body["kind"]
    # The namespace in the resource body always takes precidence
    namespace = body.get("metadata", {}).get("namespace", namespace)
    resource = getattr(api_client, "patch_namespaced_" + camel_case_to_snake_case(kind))(
        name=name, body=body, namespace=namespace
    )
    assert wait_for(
        p(has_resource, name, kind, api_client, namespace=namespace), timeout_seconds=timeout
    ), 'timed out waiting for %s "%s" to be patched!' % (kind, name)
    return resource


def delete_resource(name, kind, api_client, namespace="default", timeout=K8S_DELETE_TIMEOUT):
    if not has_resource(name, kind, api_client, namespace=namespace):
        return
    getattr(api_client, "delete_namespaced_" + camel_case_to_snake_case(kind))(
        name=name,
        body=kube_client.V1DeleteOptions(grace_period_seconds=0, propagation_policy="Background"),
        namespace=namespace,
    )
    assert wait_for(
        lambda: not has_resource(name, kind, api_client, namespace=namespace), timeout
    ), 'timed out waiting for %s "%s" to be deleted!' % (kind, name)


def has_configmap(name, namespace="default"):
    return has_resource(name, "ConfigMap", kube_client.CoreV1Api(), namespace)


def create_configmap(body, namespace=None, timeout=K8S_CREATE_TIMEOUT):
    return create_resource(body, kube_client.CoreV1Api(), namespace=namespace, timeout=timeout)


def patch_configmap(body, namespace=None, timeout=K8S_CREATE_TIMEOUT):
    return patch_resource(body, kube_client.CoreV1Api(), namespace=namespace, timeout=timeout)


def delete_configmap(name, namespace="default", timeout=K8S_DELETE_TIMEOUT):
    return delete_resource(name, "ConfigMap", kube_client.CoreV1Api(), namespace=namespace, timeout=timeout)


def has_deployment(name, namespace="default"):
    api = kube_client.ExtensionsV1beta1Api()
    try:
        api.read_namespaced_deployment(name, namespace=namespace)
        return True
    except ApiException as e:
        if e.status == 404:
            return False
        raise


def deployment_is_ready(name, replicas, namespace="default"):
    api = kube_client.ExtensionsV1beta1Api()
    if not has_deployment(name, namespace=namespace):
        return False
    if api.read_namespaced_deployment_status(name, namespace=namespace).status.ready_replicas == replicas:
        return True
    return False


def wait_for_deployment(deployment, minikube_container, timeout):
    """
    Waits for all of the ports specified in a deployment pod spec to be open
    for connections.

    :param minikube_container: A Docker container where minikube is running.
      The port checks will be performed from this container
    """
    name = deployment["metadata"]["name"]
    replicas = deployment["spec"]["replicas"]
    namespace = deployment["metadata"]["namespace"]

    assert wait_for(p(deployment_is_ready, name, replicas, namespace=namespace), timeout_seconds=timeout), (
        'timed out waiting for deployment "%s" to be ready!' % name
    )

    try:
        containers = deployment["spec"]["template"]["spec"]["containers"]
    except KeyError:
        containers = []

    for cont in containers:
        for port_spec in cont["ports"]:
            port = int(port_spec["containerPort"])
            for pod in get_all_pods_starting_with_name(name, namespace=namespace):
                assert wait_for(
                    p(pod_port_open, minikube_container, pod.status.pod_ip, port), timeout_seconds=timeout
                ), "timed out waiting for port %d for pod %s to be ready!" % (port, pod.metadata.name)


def delete_deployment(name, namespace="default", timeout=K8S_DELETE_TIMEOUT):
    if not has_deployment(name, namespace=namespace):
        return
    api = kube_client.ExtensionsV1beta1Api()
    api.delete_namespaced_deployment(
        name=name,
        body=kube_client.V1DeleteOptions(grace_period_seconds=0, propagation_policy="Background"),
        namespace=namespace,
    )
    assert wait_for(lambda: not has_deployment(name, namespace=namespace), timeout), (
        'timed out waiting for deployment "%s" to be deleted!' % name
    )


def has_daemonset(name, namespace="default"):
    api = kube_client.ExtensionsV1beta1Api()
    try:
        api.read_namespaced_daemon_set(name, namespace=namespace)
        return True
    except ApiException as e:
        if e.status == 404:
            return False
        raise


def daemonset_is_ready(name, namespace="default"):
    api = kube_client.ExtensionsV1beta1Api()
    if not has_daemonset(name, namespace=namespace):
        return False
    if api.read_namespaced_daemon_set_status(name, namespace=namespace).status.number_ready > 0:
        return True
    return False


def create_daemonset(body, namespace=None, timeout=K8S_CREATE_TIMEOUT):
    api = kube_client.ExtensionsV1beta1Api()
    name = body["metadata"]["name"]
    body["apiVersion"] = "extensions/v1beta1"
    if namespace:
        body["metadata"]["namespace"] = namespace
    else:
        try:
            namespace = body["metadata"]["namespace"]
        except KeyError:
            namespace = "default"
    if not has_namespace(namespace):
        create_namespace(namespace)
    daemonset = api.create_namespaced_daemon_set(body=body, namespace=namespace)
    assert wait_for(p(daemonset_is_ready, name, namespace=namespace), timeout_seconds=timeout), (
        'timed out waiting for daemonset "%s" to be ready!' % name
    )
    return daemonset


def delete_daemonset(name, namespace="default", timeout=K8S_DELETE_TIMEOUT):
    if not has_daemonset(name, namespace=namespace):
        return
    api = kube_client.ExtensionsV1beta1Api()
    api.delete_namespaced_daemon_set(
        name=name,
        body=kube_client.V1DeleteOptions(grace_period_seconds=0, propagation_policy="Background"),
        namespace=namespace,
    )
    assert wait_for(lambda: not has_daemonset(name, namespace=namespace), timeout), (
        'timed out waiting for daemonset "%s" to be deleted!' % name
    )


def exec_pod_command(name, command, namespace="default"):
    api = kube_client.CoreV1Api()
    pods = get_all_pods_starting_with_name(name, namespace=namespace)
    assert len(pods) == 1, 'multiple pods found with name "%s"!\n%s' % (
        name,
        "\n".join([p.metadata.name for p in pods]),
    )
    return api.connect_post_namespaced_pod_exec(name=pods[0].metadata.name, command=command, namespace=namespace)


def get_pod_logs(name, namespace="default"):
    api = kube_client.CoreV1Api()
    pods = get_all_pods_starting_with_name(name, namespace=namespace)
    assert len(pods) == 1, 'multiple pods found with name "%s"!\n%s' % (
        name,
        "\n".join([p.metadata.name for p in pods]),
    )
    return api.read_namespaced_pod_log(name=pods[0].metadata.name, namespace=namespace)


def get_all_pods(namespace=None):
    """
    Args:
    namespace (str or None): Kubernetes namespace of pods

    Returns:
    If `namespace` is None, returns list of all pods in the cluster.
    Otherwise, returns list of pods within `namespace`.
    """
    api = kube_client.CoreV1Api()
    all_pods = api.list_pod_for_all_namespaces(watch=False)
    pods = []
    for pod in all_pods.items:
        if not namespace or pod.metadata.namespace == namespace:
            pods.append(pod)
    return pods


def get_all_pods_starting_with_name(name, namespace=None):
    """
    Args:
    name (str):              Prefix of pods to search for
    namespace (str or None): Kubernetes namespace of pods

    Returns:
    If `namespace` is None, returns list of all pods in the cluster.
    Otherwise, returns list of pods within `namespace`.
    """
    pods = []
    for pod in get_all_pods(namespace=namespace):
        if pod.metadata.name.startswith(name):
            pods.append(pod)
    return pods


def has_pod(name, namespace=None):
    """
    Args:
    name (str):              Name of pods to search for
    namespace (str or None): Kubernetes namespace of pods


    Returns:
    If `namespace` is None, returns True/False if any pod contains `name`.
    Otherwise, returns True/False if any pod within `namespace` contains `name`.
    """
    for pod in get_all_pods(namespace=namespace):
        if name in pod.metadata.name:
            return True
    return False


def all_pods_have_ips(namespace="default"):
    """
    Args:
    namespace (str or None): Kubernetes namespace of pods

    Returns:
    If `namespace` is None, returns True/False if all pods have IPs.
    Otherwise, returns True/False if all pods in `namespace` have IPs.
    """
    pods = get_all_pods(namespace=namespace)
    if not pods:
        return False
    ips = 0
    for pod in pods:
        if not pod.status.pod_ip:
            return False
        ips += 1
    if ips == len(pods):
        return True
    return False


def get_all_logs(minikube):
    """
    Args:
    minikube (Minikube): Minikube instance

    Returns:
    String containing:
    - the output from 'agent-status'
    - the agent container logs
    - the output from 'minikube logs'
    - the minikube container logs
    - the status of all pods
    """
    try:
        agent_status = minikube.agent.get_status()
    except:  # noqa pylint: disable=bare-except
        agent_status = ""

    try:
        agent_container_logs = minikube.agent.get_container_logs()
    except:  # noqa pylint: disable=bare-except
        agent_container_logs = ""

    try:
        _, output = minikube.container.exec_run("minikube logs")
        minikube_logs = output.decode("utf-8").strip()
    except:  # noqa pylint: disable=bare-except
        minikube_logs = ""

    try:
        minikube_container_logs = minikube.get_container_logs()
    except:  # noqa pylint: disable=bare-except
        minikube_container_logs = ""

    try:
        pods_status = ""
        for pod in get_all_pods():
            pods_status += "%s\t%s\t%s\n" % (pod.status.pod_ip, pod.metadata.namespace, pod.metadata.name)
        pods_status = pods_status.strip()
    except:  # noqa pylint: disable=bare-except
        pods_status = ""

    return (
        "AGENT STATUS:\n%s\n\n"
        "AGENT CONTAINER LOGS:\n%s\n\n"
        "MINIKUBE LOGS:\n%s\n\n"
        "MINIKUBE CONTAINER LOGS:\n%s\n\n"
        "PODS STATUS:\n%s" % (agent_status, agent_container_logs, minikube_logs, minikube_container_logs, pods_status)
    )


def has_docker_image(client, name, tag=None):
    try:
        if tag:
            client.images.get(name + ":" + tag)
        else:
            client.images.get(name)
        return True
    except docker.errors.ImageNotFound:
        return False


def container_is_running(client, name):
    try:
        cont = client.containers.get(name)
        cont.reload()
        if cont.status.lower() != "running":
            return False
        return container_ip(cont)
    except docker.errors.NotFound:
        return False
    except docker.errors.APIError as e:
        if "is not running" in str(e):
            return False
        raise


def pod_port_open(container, host, port):
    exit_code, _ = container.exec_run("nc -z %s %d" % (host, port))
    return exit_code == 0


def get_free_port():
    sock = socket.socket()
    sock.bind(("", 0))
    return sock.getsockname()[1]


def get_discovery_rule(yaml_file, observer, namespace="", container_index=0):
    """
    Generate container discovery rule based on yaml_file.

    Args:
    yaml_file (str):       Path to K8S deployment yaml.
    observer (str):        K8S observer type (e.g. k8s-api or k8s-kubelet).
    namespace (str):       K8S namespace.
    container_index (int): Index of the container in yaml_file to generate the discovery rule for.

    Returns:
    Discovery rule (str) that can be used for monitor configuration.
    The rule will include the following endpoint variables:
    - container_state (should always match "running")
    - discovered_by (should match the observer argument)
    - container_name
    - container_names (should include container_name)
    - container_image
    - container_labels with Contains and Get (if defined in the yaml_file pod spec)
    - port (if containerPort is defined in the yaml_file pod spec)
    - network_port (if containerPort is defined in the yaml_file pod_spec)
    - private_port (if containerPort is defined in the yaml_file pod_spec)
    - kubernetes_namespace (if the namespace argument is not empty or None)
    """
    name = None
    image = None
    ports = []
    labels = []
    with open(yaml_file, "r") as fd:
        for doc in yaml.load_all(fd.read()):
            if doc["kind"] == "Deployment":
                container = doc["spec"]["template"]["spec"]["containers"][container_index]
                name = container["name"]
                image = container["image"]
                try:
                    ports = [p["containerPort"] for p in container["ports"]]
                except KeyError:
                    ports = []
                try:
                    labels = doc["spec"]["template"]["metadata"]["labels"]
                except KeyError:
                    labels = []
    assert name, "failed to get container name from %s!" % yaml_file
    assert image, "failed to get container image from %s!" % yaml_file
    rule = 'container_state == "running"'
    rule += ' && discovered_by == "%s"' % observer
    rule += ' && container_name == "%s"' % name
    rule += ' && "%s" in container_names' % name
    rule += ' && container_image == "%s"' % image
    if labels:
        for key, value in labels.items():
            rule += ' && Contains(container_labels, "%s")' % key
            rule += ' && Get(container_labels, "%s") == "%s"' % (key, value)
    if ports:
        rule += " && ((port == %s" % ports[0]
        rule += " && network_port == %s" % ports[0]
        rule += " && private_port == %s)" % ports[0]
        if len(ports) > 1:
            for port in ports[1:]:
                rule += " || (port == %s" % port
                rule += " && network_port == %s" % port
                rule += " && private_port == %s)" % port
        rule += ")"
    if namespace:
        rule += ' && kubernetes_namespace == "%s"' % namespace
    return rule

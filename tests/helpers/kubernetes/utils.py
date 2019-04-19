import os
import re
from functools import partial as p
from pathlib import Path

import yaml
from kubernetes import client as kube_client
from kubernetes.client.rest import ApiException
from kubernetes.stream import stream

from tests.helpers.assertions import has_any_metric_or_dim
from tests.helpers.util import get_observer_dims_from_selfdescribe, wait_for

K8S_CREATE_TIMEOUT = int(os.environ.get("K8S_CREATE_TIMEOUT", 300))
K8S_DELETE_TIMEOUT = 10


def run_k8s_monitors_test(  # pylint: disable=too-many-locals,too-many-arguments,dangerous-default-value
    agent_image,
    minikube,
    monitors,
    observer=None,
    namespace="default",
    yamls=None,
    yamls_timeout=K8S_CREATE_TIMEOUT,
    expected_metrics=None,
    expected_dims=None,
    passwords=None,
    test_timeout=60,
):
    """
    Wrapper function for K8S setup and tests within minikube for monitors.

    Setup includes starting the fake backend, creating K8S deployments, and smart agent configuration/deployment
    within minikube.

    Tests include waiting for at least one metric and/or dimension from the expected_metrics and expected_dims args,
    and checking for cleartext passwords in the output from the agent status and agent container logs.

    Required Args:
    agent_image (dict):                    Dict object from the agent_image fixture
    minikube (Minikube):                   Minkube object from the minikube fixture
    monitors (str, dict, or list of dict): YAML-based definition of monitor(s) for the smart agent agent.yaml

    Optional Args:
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
    try:
        monitors = yaml.load(monitors)
    except AttributeError:
        pass
    if isinstance(monitors, dict):
        monitors = [monitors]
    assert isinstance(monitors, list), "unknown type/defintion for monitors:\n%s\n" % monitors
    if not yamls:
        yamls = []
    if not expected_metrics:
        expected_metrics = set()
    if not expected_dims:
        expected_dims = set()
    if observer:
        expected_dims = expected_dims.union(get_observer_dims_from_selfdescribe(observer))
    expected_dims = expected_dims.union({"kubernetes_cluster"})
    if passwords is None:
        passwords = ["testing123"]

    with minikube.create_resources(yamls, namespace=namespace, timeout=yamls_timeout):
        with minikube.run_agent(agent_image, observer=observer, monitors=monitors, namespace=namespace) as [
            agent,
            backend,
        ]:
            assert wait_for(
                p(has_any_metric_or_dim, backend, expected_metrics, expected_dims), timeout_seconds=test_timeout
            ), "timed out waiting for metrics in %s with any dimensions in %s!" % (expected_metrics, expected_dims)
            assert all(
                [p not in agent.get_status() for p in passwords]
            ), "cleartext password(s) found in agent-status output!"
            assert all(
                [p not in agent.get_logs() for p in passwords]
            ), "cleartext password(s) found in agent container logs!"


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
        namespace = body["metadata"].get("namespace", "default")
    if not has_namespace(namespace):
        create_namespace(namespace)
    serviceaccount = api.create_namespaced_service_account(body=body, namespace=namespace)
    assert wait_for(p(has_serviceaccount, name, namespace=namespace), timeout_seconds=timeout), (
        'timed out waiting for service account "%s" to be created!' % name
    )
    return serviceaccount


def has_clusterrole(name):
    api = kube_client.RbacAuthorizationV1beta1Api()
    try:
        api.read_cluster_role(name)
        return True
    except ApiException as e:
        if e.status == 404:
            return False
        raise


def create_clusterrole(body, timeout=K8S_CREATE_TIMEOUT):
    api = api_client_from_version(body["apiVersion"])
    name = body["metadata"]["name"]
    clusterrole = api.create_cluster_role(body=body)
    assert wait_for(p(has_clusterrole, name), timeout_seconds=timeout), (
        'timed out waiting for cluster role "%s" to be created!' % name
    )
    return clusterrole


def has_clusterrolebinding(name):
    api = kube_client.RbacAuthorizationV1beta1Api()
    try:
        api.read_cluster_role_binding(name)
        return True
    except ApiException as e:
        if e.status == 404:
            return False
        raise


def create_clusterrolebinding(body, timeout=K8S_CREATE_TIMEOUT):
    api = api_client_from_version(body["apiVersion"])
    name = body["metadata"]["name"]
    clusterrolebinding = api.create_cluster_role_binding(body=body)
    assert wait_for(p(has_clusterrolebinding, name), timeout_seconds=timeout), (
        'timed out waiting for cluster role binding "%s" to be created!' % name
    )
    return clusterrolebinding


def api_client_from_version(api_version):
    return {
        "v1": kube_client.CoreV1Api(),
        "extensions/v1beta1": kube_client.ExtensionsV1beta1Api(),
        "rbac.authorization.k8s.io/v1beta1": kube_client.RbacAuthorizationV1beta1Api(),
        "rbac.authorization.k8s.io/v1": kube_client.RbacAuthorizationV1Api(),
    }[api_version]


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
    api = api_client_from_version(body["apiVersion"])
    return create_resource(body, api, namespace=namespace, timeout=timeout)


def patch_configmap(body, namespace=None, timeout=K8S_CREATE_TIMEOUT):
    api = api_client_from_version(body["apiVersion"])
    return patch_resource(body, api, namespace=namespace, timeout=timeout)


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


def deployment_is_ready(name, namespace="default"):
    api = kube_client.ExtensionsV1beta1Api()
    if not has_deployment(name, namespace=namespace):
        return False
    status = api.read_namespaced_deployment_status(name, namespace=namespace).status
    if status and status.ready_replicas and status.available_replicas:
        return status.ready_replicas == status.available_replicas
    return False


def delete_deployment(name, namespace="default", timeout=K8S_DELETE_TIMEOUT):
    if not has_deployment(name, namespace=namespace):
        return
    api = kube_client.ExtensionsV1beta1Api()
    api.delete_namespaced_deployment(
        name=name,
        body=kube_client.V1DeleteOptions(grace_period_seconds=0, propagation_policy="Background"),
        namespace=namespace,
    )
    assert wait_for(lambda: not has_deployment(name, namespace=namespace), timeout_seconds=timeout), (
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
    status = api.read_namespaced_daemon_set_status(name, namespace=namespace).status
    if status and status.number_ready and status.number_ready:
        return status.number_ready == status.number_available
    return False


def create_daemonset(body, namespace=None, timeout=K8S_CREATE_TIMEOUT):
    name = body["metadata"]["name"]
    api = api_client_from_version(body["apiVersion"])
    daemonset = create_resource(body, api, namespace=namespace, timeout=timeout)
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
    assert wait_for(lambda: not has_daemonset(name, namespace=namespace), timeout_seconds=timeout), (
        'timed out waiting for daemonset "%s" to be deleted!' % name
    )


def get_pods_by_labels(labels, namespace="default"):
    """
    Returns a list of pods matching `labels` within `namespace`.
    `labels` should be a comma-separated string of "key=value" pairs.
    """
    api = kube_client.CoreV1Api()
    return api.list_namespaced_pod(namespace, label_selector=labels).items


def exec_pod_command(name, command, namespace="default"):
    api = kube_client.CoreV1Api()
    pod = api.read_namespaced_pod(name, namespace=namespace)
    return stream(
        api.connect_get_namespaced_pod_exec,
        name=pod.metadata.name,
        command=command,
        namespace=namespace,
        stderr=True,
        stdin=False,
        stdout=True,
        tty=False,
        _preload_content=True,
        _request_timeout=5,
    ).strip()


def get_pod_logs(name, namespace="default"):
    api = kube_client.CoreV1Api()
    pods = get_all_pods_starting_with_name(name, namespace=namespace)
    assert pods, "no pods found with name '%s'" % name
    logs = ""
    for pod in pods:
        if pod.status.container_statuses:
            for container in pod.status.container_statuses:
                logs += "%s container log:\n" % container.name
                try:
                    logs += api.read_namespaced_pod_log(
                        name=pod.metadata.name, container=container.name, namespace=namespace
                    ).strip()
                except ApiException as e:
                    logs += "failed to get log:\n%s" % str(e).strip()
                logs += "\n"
    return logs.strip()


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


def pod_is_ready(name, namespace="default"):
    api = kube_client.CoreV1Api()
    pod = api.read_namespaced_pod(name, namespace=namespace)
    return pod.status.phase.lower() == "running" and all(
        [container.ready for container in pod.status.container_statuses]
    )


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


def get_metrics(dir_, name="metrics.txt"):
    """Returns set of metrics from file"""
    return {m.strip() for m in (Path(dir_) / name).read_text().splitlines() if len(m.strip()) > 0}

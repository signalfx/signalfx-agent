import subprocess
from contextlib import contextmanager
from functools import partial as p
from pathlib import Path

import yaml
from kubernetes import client as kube_client
from tests.helpers.util import pull_from_reader_in_background, wait_for

from .utils import get_pod_logs, pod_is_ready

SCRIPT_DIR = Path(__file__).parent.resolve()


@contextmanager
def deploy_fake_backend_proxy_pod(namespace):
    """
    Deploys a socat pod named "fake-backend" that is ready to be used to
    tunnel datapoints back to this process.
    """
    corev1 = kube_client.CoreV1Api()
    pod_yaml = Path(SCRIPT_DIR / "tunnel/pod.yaml").read_bytes()
    pod = corev1.create_namespaced_pod(body=yaml.safe_load(pod_yaml), namespace=namespace)
    name = pod.metadata.name
    try:
        assert wait_for(p(pod_is_ready, name, namespace=namespace), timeout_seconds=45)
        yield corev1.read_namespaced_pod(name, namespace=namespace).status.pod_ip
    finally:
        print("Fake backend proxy logs: %s" % (get_pod_logs(name, namespace=namespace)))
        corev1.delete_namespaced_pod(
            name,
            namespace=namespace,
            body=kube_client.V1DeleteOptions(grace_period_seconds=0, propagation_policy="Background"),
        )


def start_tunneling_fake_service(local_host, local_port, namespace, kube_config_path, context):
    """
    Run the client.sh script that sets up a remote tunnel from the cluster back
    to the fake backend components running locally.
    """
    proc = subprocess.Popen(
        ["/bin/bash", f"{SCRIPT_DIR}/tunnel/client.sh"],
        env={
            "KUBECONFIG": kube_config_path,
            "KUBE_CONTEXT": context or "",
            "LOCAL_HOST": local_host,
            "LOCAL_PORT": str(local_port),
            "REMOTE_PORT": str(local_port),
            "NAMESPACE": namespace,
        },
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )

    get_output = pull_from_reader_in_background(proc.stdout)

    def term_func():
        proc.terminate()
        proc.wait()

    return term_func, get_output

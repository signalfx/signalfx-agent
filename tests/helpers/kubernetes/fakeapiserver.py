import warnings
from contextlib import contextmanager

import urllib3.exceptions
from kubernetes import client

from tests.helpers.assertions import tcp_socket_open
from tests.helpers.util import container_ip, run_service, wait_for
from tests.paths import REPO_ROOT_DIR


@contextmanager
def fake_k8s_api_server(print_logs=False):
    with run_service(
        "fakek8s", print_logs=print_logs, path=REPO_ROOT_DIR, dockerfile="./test-services/fakek8s/Dockerfile"
    ) as fakek8s_cont:
        ipaddr = container_ip(fakek8s_cont)
        conf = client.Configuration()
        conf.host = f"https://{ipaddr}:8443"
        conf.verify_ssl = False

        assert wait_for(lambda: tcp_socket_open(ipaddr, 8443)), "fake k8s never opened port"
        warnings.filterwarnings("ignore", category=urllib3.exceptions.InsecureRequestWarning)

        yield [client.ApiClient(conf), {"KUBERNETES_SERVICE_HOST": ipaddr, "KUBERNETES_SERVICE_PORT": "8443"}]

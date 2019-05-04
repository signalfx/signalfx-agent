import io
import os
import re
import socket
import subprocess
import threading
import time
from contextlib import contextmanager
from functools import partial as p
from typing import Dict, List

import docker
import netifaces as ni
import yaml
from tests.helpers.assertions import regex_search_matches_output
from tests.paths import SELFDESCRIBE_JSON, TEST_SERVICES_DIR

DEFAULT_TIMEOUT = int(os.environ.get("DEFAULT_TIMEOUT", 30))
DOCKER_API_VERSION = "1.34"
STATSD_RE = re.compile(r"SignalFx StatsD monitor: Listening on host & port udp:\[::\]:([0-9]*)")


def get_docker_client():
    return docker.from_env(version=DOCKER_API_VERSION)


def has_docker_image(client, name):
    return name in [t for image in client.images.list() for t in image.tags]


def assert_wait_for(test, timeout_seconds=DEFAULT_TIMEOUT, interval_seconds=0.2, on_fail=None):
    """
    Runs `wait_for` but raises an assertion if it fails, optionally calling
    `on_fail` before raising an AssertionError
    """
    if not wait_for(test, timeout_seconds, interval_seconds):
        if on_fail:
            on_fail()

        raise AssertionError("test '%s' still failng after %d seconds" % (test, timeout_seconds))


def wait_for(test, timeout_seconds=DEFAULT_TIMEOUT, interval_seconds=0.2):
    """
    Repeatedly calls the test function for timeout_seconds until either test
    returns a truthy value, at which point the function returns True -- or the
    timeout is exceeded, at which point it will return False.
    """
    start = time.time()
    while True:
        if test():
            return True
        if time.time() - start > timeout_seconds:
            return False
        time.sleep(interval_seconds)


def ensure_always(test, timeout_seconds=DEFAULT_TIMEOUT, interval_seconds=0.2):
    """
    Repeatedly calls the given test.  If it ever returns false before the timeout
    given is completed, returns False, otherwise True.
    """
    start = time.time()
    while True:
        if not test():
            return False
        if time.time() - start > timeout_seconds:
            return True
        time.sleep(interval_seconds)


def ensure_never(test, timeout_seconds=DEFAULT_TIMEOUT):
    """
    Repeatedly calls the given test.  If it ever returns true before the timeout
    given is completed, returns False, otherwise True.
    """
    start = time.time()
    while True:
        if test():
            return False
        if time.time() - start > timeout_seconds:
            return True
        time.sleep(0.2)


def print_lines(msg):
    """
    Print each line separately to make it easier to read in pytest output
    """
    for line in msg.splitlines():
        print(line)


def container_ip(container):
    container.reload()
    return container.attrs["NetworkSettings"]["IPAddress"]


# Ensure a unique internal status server host address.  This supports up to
# 255 concurrent agents on the same pytest worker process, and up to 255
# pytest workers, which should be plenty
def get_unique_localhost():
    worker = int(re.sub(r"\D", "", os.environ.get("PYTEST_XDIST_WORKER", "0")))
    get_unique_localhost.counter += 1
    return "127.%d.%d.0" % (worker, get_unique_localhost.counter % 255)


get_unique_localhost.counter = 0


@contextmanager
def run_subprocess(command: List[str], env: Dict[any, any] = None):
    # subprocess on Windows has a bug where it doesn't like Path.
    proc = subprocess.Popen([str(c) for c in command], env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)

    get_output = pull_from_reader_in_background(proc.stdout)

    try:
        yield [get_output, proc.pid]
    finally:
        proc.terminate()
        proc.wait(15)


@contextmanager
def run_container(image_name, wait_for_ip=True, print_logs=True, **kwargs):
    client = get_docker_client()
    container = retry(lambda: client.containers.run(image_name, detach=True, **kwargs), docker.errors.DockerException)

    def has_ip_addr():
        container.reload()
        return container.attrs["NetworkSettings"]["IPAddress"]

    if wait_for_ip:
        wait_for(has_ip_addr, timeout_seconds=5)
    try:
        yield container
    finally:
        try:
            if print_logs:
                print_lines(
                    "Container %s/%s logs:\n%s" % (image_name, container.name, container.logs().decode("utf-8"))
                )
            container.remove(force=True, v=True)
        except docker.errors.NotFound:
            pass


@contextmanager
def run_service(service_name, buildargs=None, print_logs=True, path=None, dockerfile="./Dockerfile", **kwargs):
    if buildargs is None:
        buildargs = {}
    if path is None:
        path = os.path.join(TEST_SERVICES_DIR, service_name)

    client = get_docker_client()
    image, _ = retry(
        lambda: client.images.build(path=str(path), dockerfile=dockerfile, rm=True, forcerm=True, buildargs=buildargs),
        docker.errors.BuildError,
    )
    with run_container(image.id, print_logs=print_logs, **kwargs) as cont:
        yield cont


def get_monitor_metrics_from_selfdescribe(monitor, json_path=SELFDESCRIBE_JSON):
    metrics = set()
    with open(json_path, "r", encoding="utf-8") as fd:
        doc = yaml.safe_load(fd.read())
        for mon in doc["Monitors"]:
            if monitor == mon["monitorType"] and "metrics" in mon.keys() and mon["metrics"]:
                metrics = set(mon["metrics"].keys())
                break
    return metrics


def get_monitor_dims_from_selfdescribe(monitor, json_path=SELFDESCRIBE_JSON):
    dims = set()
    with open(json_path, "r", encoding="utf-8") as fd:
        doc = yaml.safe_load(fd.read())
        for mon in doc["Monitors"]:
            if monitor == mon["monitorType"] and "dimensions" in mon.keys() and mon["dimensions"]:
                dims = set(mon["dimensions"].keys())
                break
    return dims


def get_observer_dims_from_selfdescribe(observer, json_path=SELFDESCRIBE_JSON):
    dims = set()
    with open(json_path, "r", encoding="utf-8") as fd:
        doc = yaml.safe_load(fd.read())
        for obs in doc["Observers"]:
            if observer == obs["observerType"] and "dimensions" in obs.keys() and obs["dimensions"]:
                dims = set(obs["dimensions"].keys())
                break
    return dims


def get_host_ip():
    gws = ni.gateways()
    interface = gws["default"][ni.AF_INET][1]
    return ni.ifaddresses(interface)[ni.AF_INET][0]["addr"]


def send_udp_message(host, port, msg):
    """
    Send a datagram to the given host/port
    """
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)  # Internet  # UDP
    sock.sendto(msg.encode("utf-8"), (host, port))


def retry(function, exception, max_attempts=5, interval_seconds=5):
    """
    Retry function up to max_attempts if exception is caught
    """
    for attempt in range(max_attempts):
        try:
            return function()
        except exception as e:
            assert attempt < (max_attempts - 1), "%s failed after %d attempts!\n%s" % (function, max_attempts, str(e))
        time.sleep(interval_seconds)


def get_statsd_port(agent):
    """
    Discover an open port of running StatsD monitor
    """
    assert wait_for(p(regex_search_matches_output, agent.get_output, STATSD_RE.search))
    regex_results = STATSD_RE.search(agent.output)
    return int(regex_results.groups()[0])


def pull_from_reader_in_background(reader):
    output = io.BytesIO()

    def pull_output():
        while True:
            # If any output is waiting, grab it.
            try:
                byt = reader.read(1)
            except OSError:
                return
            if not byt:
                return
            if isinstance(byt, str):
                byt = byt.encode("utf-8")
            output.write(byt)

    threading.Thread(target=pull_output).start()

    def get_output():
        return output.getvalue().decode("utf-8")

    return get_output

import io
import os
import re
import socket
import subprocess
import sys
import tempfile
import threading
import time
from contextlib import contextmanager
from typing import Dict, List

import docker
import netifaces as ni
import yaml

from . import fake_backend
from .formatting import print_dp_or_event
from .internalmetrics import InternalMetricsClient
from .profiling import PProfClient

PROJECT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
if sys.platform == "win32":
    AGENT_BIN = os.environ.get("AGENT_BIN", os.path.join(PROJECT_DIR, "..", "signalfx-agent.exe"))
else:
    AGENT_BIN = os.environ.get("AGENT_BIN", "/bundle/bin/signalfx-agent")
REPO_ROOT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
BUNDLE_DIR = os.environ.get("BUNDLE_DIR", "/bundle")
TEST_SERVICES_DIR = os.environ.get("TEST_SERVICES_DIR", "/test-services")
DEFAULT_TIMEOUT = os.environ.get("DEFAULT_TIMEOUT", 30)
DOCKER_API_VERSION = "1.34"
SELFDESCRIBE_JSON = os.path.join(PROJECT_DIR, "../selfdescribe.json")


def get_docker_client():
    return docker.from_env(version=DOCKER_API_VERSION)


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


@contextmanager
def run_agent(config_text, debug=True, profile=False, extra_env=None, internal_metrics=False, backend_options=None):
    host = get_unique_localhost()

    with fake_backend.start(host, **(backend_options or {})) as fake_services:
        with run_agent_with_fake_backend(
            config_text, fake_services, debug=debug, host=host, profile=profile, extra_env=extra_env
        ) as [_, get_output, setup_conf, conf]:
            to_yield = [fake_services, get_output, setup_conf]
            if profile:
                to_yield += [PProfClient(conf["profilingHost"], conf.get("profilingPort", 6060))]
            if internal_metrics:
                to_yield += [InternalMetricsClient(conf["internalStatusHost"], conf["internalStatusPort"])]
            yield to_yield


@contextmanager
def run_agent_with_fake_backend(config_text, fake_services, debug=True, host=None, profile=False, extra_env=None):
    with tempfile.TemporaryDirectory() as run_dir:
        config_path = os.path.join(run_dir, "agent.yaml")

        conf = render_config(config_text, config_path, fake_services, debug=debug, host=host, profile=profile)

        agent_env = {**os.environ.copy(), **(extra_env or {})}

        with run_subprocess(
            [AGENT_BIN, "-config", config_path] + (["-debug"] if debug else []), env=agent_env
        ) as get_output:
            try:
                yield [fake_services, get_output, lambda c: render_config(c, config_path, fake_services), conf]
            finally:
                print("\nAgent output:")
                print_lines(get_output())
                if debug:
                    print("\nDatapoints received:")
                    for dp in fake_services.datapoints:
                        print_dp_or_event(dp)
                    print("\nEvents received:")
                    for event in fake_services.events:
                        print_dp_or_event(event)


@contextmanager
def run_subprocess(command: List[str], env: Dict[any, any] = None):
    proc = subprocess.Popen(command, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)

    output = io.BytesIO()

    def pull_output():
        while True:
            # If any output is waiting, grab it.
            byt = proc.stdout.read(1)
            if not byt:
                return
            output.write(byt)

    def get_output():
        return output.getvalue().decode("utf-8")

    threading.Thread(target=pull_output).start()

    try:
        yield get_output
    finally:
        proc.terminate()
        proc.wait(15)


# Ensure a unique internal status server host address.  This supports up to
# 255 concurrent agents on the same pytest worker process, and up to 255
# pytest workers, which should be plenty
def get_unique_localhost():
    worker = int(re.sub(r"\D", "", os.environ.get("PYTEST_XDIST_WORKER", "0")))
    get_unique_localhost.counter += 1
    return "127.%d.%d.0" % (worker, get_unique_localhost.counter % 255)


get_unique_localhost.counter = 0


def render_config(config_text, path, fake_services, debug=True, host=None, profile=False) -> Dict:
    if config_text is None and os.path.isfile(path):
        return path

    conf = yaml.load(config_text)

    run_dir = os.path.dirname(path)

    if conf.get("intervalSeconds") is None:
        conf["intervalSeconds"] = 3

    if conf.get("signalFxAccessToken") is None:
        conf["signalFxAccessToken"] = "testing123"

    conf["ingestUrl"] = fake_services.ingest_url
    conf["apiUrl"] = fake_services.api_url

    if host is None:
        host = get_unique_localhost()

    conf["internalStatusHost"] = host
    conf["internalStatusPort"] = 8095
    if profile:
        conf["profiling"] = True
        conf["profilingHost"] = host

    conf["logging"] = dict(level="debug" if debug else "info")

    conf["collectd"] = conf.get("collectd", {})
    conf["collectd"]["configDir"] = os.path.join(run_dir, "collectd")
    conf["collectd"]["logLevel"] = "info"

    conf["configSources"] = conf.get("configSources", {})
    conf["configSources"]["file"] = conf["configSources"].get("file", {})
    conf["configSources"]["file"]["pollRateSeconds"] = 1

    with open(path, "w") as fd:
        print("CONFIG: %s\n%s" % (path, conf))
        fd.write(yaml.dump(conf))

    return conf


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
        lambda: client.images.build(path=path, dockerfile=dockerfile, rm=True, forcerm=True, buildargs=buildargs),
        docker.errors.BuildError,
    )
    with run_container(image.id, print_logs=print_logs, **kwargs) as cont:
        yield cont


def get_monitor_metrics_from_selfdescribe(monitor, json_path=SELFDESCRIBE_JSON):
    metrics = set()
    with open(json_path, "r", encoding="utf-8") as fd:
        doc = yaml.load(fd.read())
        for mon in doc["Monitors"]:
            if monitor == mon["monitorType"] and "metrics" in mon.keys() and mon["metrics"]:
                metrics = {metric["name"] for metric in mon["metrics"]}
                break
    return metrics


def get_monitor_metrics_from_metadata_yaml(monitor_package_path, mon_type=None):
    with open(os.path.join(REPO_ROOT_DIR, monitor_package_path, "metadata.yaml"), "r", encoding="utf-8") as fd:
        doc = yaml.safe_load(fd.read())
        if len(doc) == 1:
            return doc[0].get("metrics")

        if mon_type is None:
            raise ValueError(
                "mon_type kwarg must be provided when there is more than one monitor in a metadata.yaml file"
            )
        for monitor in doc:
            if monitor["monitorType"] == mon_type:
                return monitor.get("metrics")
    return None


def get_monitor_default_metrics_list_from_metadata_yaml(monitor_package_path, mon_type=None):
    metrics = get_monitor_metrics_from_metadata_yaml(monitor_package_path, mon_type)
    ret = []
    for metric in metrics:
        if metric.get("included"):
            ret.append(metric.get("name"))
    return ret


def get_monitor_dimensions_list_from_metadata_yaml(monitor_package_path, mon_type=None):
    with open(os.path.join(REPO_ROOT_DIR, monitor_package_path, "metadata.yaml"), "r", encoding="utf-8") as fd:
        doc = yaml.safe_load(fd.read())
        out = []
        if len(doc) == 1:
            for dim in doc[0].get("dimensions"):
                out.append(dim.get("name"))
            return out

        if mon_type is None:
            raise ValueError(
                "mon_type kwarg must be provided when there is more than one monitor in a metadata.yaml file"
            )
        for monitor in doc:
            if monitor["monitorType"] == mon_type:
                for dim in monitor[0].get("dimensions"):
                    out.append(dim.get("name"))
                return out
    return out


def get_monitor_dims_from_selfdescribe(monitor, json_path=SELFDESCRIBE_JSON):
    dims = set()
    with open(json_path, "r", encoding="utf-8") as fd:
        doc = yaml.load(fd.read())
        for mon in doc["Monitors"]:
            if monitor == mon["monitorType"] and "dimensions" in mon.keys() and mon["dimensions"]:
                dims = {dim["name"] for dim in mon["dimensions"]}
                break
    return dims


def get_observer_dims_from_selfdescribe(observer, json_path=SELFDESCRIBE_JSON):
    dims = set()
    with open(json_path, "r", encoding="utf-8") as fd:
        doc = yaml.load(fd.read())
        for obs in doc["Observers"]:
            if observer == obs["observerType"] and "dimensions" in obs.keys() and obs["dimensions"]:
                dims = {dim["name"] for dim in obs["dimensions"]}
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


def get_agent_status(config_path="/etc/signalfx/agent.yaml"):
    status_proc = subprocess.Popen(
        [AGENT_BIN, "status", "-config", config_path], stdout=subprocess.PIPE, stderr=subprocess.STDOUT
    )
    return status_proc.stdout.read().decode("utf-8")


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

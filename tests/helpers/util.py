import io
import os
import select
import socket
import subprocess
import tempfile
import threading
import time
from contextlib import contextmanager

import docker
import yaml

import netifaces as ni

from . import fake_backend
from .formatting import print_dp_or_event

AGENT_BIN = os.environ.get("AGENT_BIN", "/bundle/bin/signalfx-agent")
BUNDLE_DIR = os.environ.get("BUNDLE_DIR", "/bundle")
PROJECT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
TEST_SERVICES_DIR = os.environ.get("TEST_SERVICES_DIR", "/test-services")
DEFAULT_TIMEOUT = os.environ.get("DEFAULT_TIMEOUT", 30)
DOCKER_API_VERSION = "1.34"
SELFDESCRIBE_JSON = os.path.join(PROJECT_DIR, "../selfdescribe.json")


def get_docker_client():
    return docker.from_env(version=DOCKER_API_VERSION)


def wait_for(test, timeout_seconds=DEFAULT_TIMEOUT):
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
        time.sleep(0.2)


def ensure_always(test, timeout_seconds=DEFAULT_TIMEOUT):
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
        time.sleep(0.2)


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
    return container.attrs["NetworkSettings"]["IPAddress"]


@contextmanager
def run_agent(config_text):
    with fake_backend.start() as fake_services:
        with run_agent_with_fake_backend(config_text, fake_services) as [_, get_output, setup_conf]:
            yield [fake_services, get_output, setup_conf]


@contextmanager
def run_agent_with_fake_backend(config_text, fake_services):
    with tempfile.TemporaryDirectory() as run_dir:
        config_path = os.path.join(run_dir, "agent.yaml")

        setup_config(config_text, config_path, fake_services)

        proc = subprocess.Popen([AGENT_BIN, "-config", config_path], stdout=subprocess.PIPE, stderr=subprocess.STDOUT)

        output = io.BytesIO()

        def pull_output():
            while True:
                # If any output is waiting, grab it.
                ready, _, _ = select.select([proc.stdout], [], [], 0)
                if ready:
                    byt = proc.stdout.read(1)
                    if not byt:
                        return
                    output.write(byt)

        def get_output():
            return output.getvalue().decode("utf-8")

        threading.Thread(target=pull_output).start()

        try:
            yield [fake_services, get_output, lambda c: setup_config(c, config_path, fake_services)]
        finally:
            exc = None

            proc.terminate()
            try:
                proc.wait(15)
            except subprocess.TimeoutExpired as e:
                exc = e

            print("\nAgent output:")
            print_lines(get_output())
            print("\nDatapoints received:")
            for dp in fake_services.datapoints:
                print_dp_or_event(dp)
            print("\nEvents received:")
            for event in fake_services.events:
                print_dp_or_event(event)

            if exc is not None:
                raise exc  # pylint: disable=raising-bad-type


def setup_config(config_text, path, fake_services):
    conf = yaml.load(config_text)

    run_dir = os.path.dirname(path)

    if conf.get("intervalSeconds") is None:
        conf["intervalSeconds"] = 3

    if conf.get("signalFxAccessToken") is None:
        conf["signalFxAccessToken"] = "testing123"

    conf["ingestUrl"] = fake_services.ingest_url
    conf["apiUrl"] = fake_services.api_url
    conf["internalMetricsSocketPath"] = os.path.join(run_dir, "internal.sock")
    conf["diagnosticsSocketPath"] = os.path.join(run_dir, "diagnostics.sock")
    conf["logging"] = dict(level="debug")

    conf["collectd"] = conf.get("collectd", {})
    conf["collectd"]["configDir"] = os.path.join(run_dir, "collectd")
    conf["collectd"]["logLevel"] = "info"

    conf["configSources"] = conf.get("configSources", {})
    conf["configSources"]["file"] = conf["configSources"].get("file", {})
    conf["configSources"]["file"]["pollRateSeconds"] = 1

    with open(path, "w") as fd:
        print("CONFIG: %s\n%s" % (path, conf))
        fd.write(yaml.dump(conf))


@contextmanager
def run_container(image_name, wait_for_ip=True, print_logs=True, **kwargs):
    client = get_docker_client()
    container = client.containers.run(image_name, detach=True, **kwargs)

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
                print_lines("Container %s logs:\n%s" % (image_name, container.logs().decode("utf-8")))
            container.remove(force=True, v=True)
        except docker.errors.NotFound:
            pass


@contextmanager
def run_service(service_name, name=None, buildargs=None, print_logs=True, **kwargs):
    if buildargs is None:
        buildargs = {}
    client = get_docker_client()
    image, _ = client.images.build(
        path=os.path.join(TEST_SERVICES_DIR, service_name), rm=True, forcerm=True, buildargs=buildargs
    )
    with run_container(image.id, name=name, print_logs=print_logs, **kwargs) as cont:
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

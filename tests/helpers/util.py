from contextlib import contextmanager
import docker
import inspect
import io
import os
import queue
import select
import shutil
import subprocess
import string
import tempfile
import threading
import time
import yaml
from . import fake_backend

AGENT_BIN = os.environ.get("AGENT_BIN", "/bundle/bin/signalfx-agent")
PROJECT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
TEST_SERVICES_DIR = os.environ.get("TEST_SERVICES_DIR", "/test-services")
DEFAULT_TIMEOUT = os.environ.get("DEFAULT_TIMEOUT", 20)
DOCKER_API_VERSION = "1.34"


def get_docker_client():
    return docker.from_env(version=DOCKER_API_VERSION)

# Repeatedly calls the test function for timeout_seconds until either test
# returns a truthy value, at which point the function returns True -- or the
# timeout is exceeded, at which point it will return False.
def wait_for(test, timeout_seconds=DEFAULT_TIMEOUT):
    start = time.time()
    while True:
        if test():
            return True
        if time.time() - start > timeout_seconds:
            return False
        time.sleep(0.5)


# Repeatedly calls the given test.  If it ever returns false before the timeout
# given is completed, returns False, otherwise True.
def ensure_always(test, timeout_seconds=DEFAULT_TIMEOUT):
    start = time.time()
    while True:
        if not test():
            return False
        if time.time() - start > timeout_seconds:
            return True
        time.sleep(0.5)


# Print each line separately to make it easier to read in pytest output
def print_lines(msg):
    for l in msg.splitlines():
        print(l)


def container_ip(container):
    return container.attrs["NetworkSettings"]["IPAddress"]


@contextmanager
def run_agent(config_text):
    with fake_backend.start() as fake_services:
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
                        b = proc.stdout.read(1)
                        if not b:
                            return
                        output.write(b)

            def get_output():
                return output.getvalue().decode("utf-8")

            threading.Thread(target=pull_output).start()

            try:
                yield [fake_services, get_output, lambda c: setup_config(c, config_path, fake_services)]
            finally:
                proc.terminate()
                proc.wait(10)

                print("Agent output:")
                print_lines(get_output())


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

    conf["configSources"] = conf.get("configSources", {})
    conf["configSources"]["file"] = conf["configSources"].get("file", {})
    conf["configSources"]["file"]["pollRateSeconds"] = 1

    with open(path, "w") as f:
        print("CONFIG: %s\n%s" % (path, conf))
        f.write(yaml.dump(conf))


@contextmanager
def run_container(image_name, wait_for_ip=True, **kwargs):
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
        print_lines("Container %s logs:\n%s" % (image_name, container.logs()))
        container.remove(force=True, v=True)


@contextmanager
def run_service(service_name, name=None):
    client = get_docker_client()
    image, logs = client.images.build(path=os.path.join(TEST_SERVICES_DIR, service_name), rm=True, forcerm=True)
    with run_container(image.id, name=name) as cont:
        yield cont


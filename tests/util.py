from contextlib import contextmanager
import docker
import inspect
import io
import os
import select
import shutil
import subprocess
import string
import tempfile
import time
import yaml
from . import fake_backend

AGENT_BIN = os.environ.get("AGENT_BIN", "/bundle/bin/signalfx-agent")
PROJECT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
TEST_SERVICES_DIR = os.environ.get("TEST_SERVICES_DIR", "/test-services")
DEFAULT_TIMEOUT = os.environ.get("DEFAULT_TIMEOUT", 20)


# Repeatedly calls the test function for timeout_seconds until either test
# returns a truthy value or the timeout is exceeded, at which point it will
# raise an AssertionError.
def wait_for(test, timeout_seconds=DEFAULT_TIMEOUT):
    start = time.time()
    while True:
        if test():
            return
        if time.time() - start > timeout_seconds:
            raise AssertionError("Test failed for %d seconds" % timeout_seconds)
        time.sleep(0.5)


def container_ip(container):
    return container.attrs["NetworkSettings"]["IPAddress"]


@contextmanager
def run_agent(config_text):
    with fake_backend.start() as fake_services:
        with tempfile.TemporaryDirectory() as run_dir:
            config_path = setup_config(config_text, run_dir, fake_services)
            print("CONFIG: %s\n%s" % (config_path, config_text))

            proc = subprocess.Popen([AGENT_BIN, "-config", config_path], stdout=subprocess.PIPE, stderr=subprocess.STDOUT)

            output = io.StringIO() 
            def get_output():
                # If any output is waiting, grab it.
                ready, _, _ = select.select([proc.stdout], [], [], 0)
                if ready:
                    output.write(proc.stdout.read().decode("utf-8"))

                return output.getvalue()

            try:
                yield [fake_services, get_output]
            finally:
                proc.terminate()
                proc.wait(10)

                print("Agent output:")
                for line in get_output().splitlines():
                    print(line)


def setup_config(config_text, run_dir, fake_services):
    conf = yaml.load(config_text)

    if conf.get("intervalSeconds") is None:
        conf["intervalSeconds"] = 3

    if conf.get("signalFxAccessToken") is None:
        conf["signalFxAccessToken"] = "testing123"

    conf["ingestUrl"] = fake_services.ingest_url
    conf["apiUrl"] = fake_services.api_url
    conf["internalMetricsSocketPath"] = os.path.join(run_dir, "internal.sock")
    conf["diagnosticsSocketPath"] = os.path.join(run_dir, "diagnostics.sock")

    conf["collectd"] = conf.get("collectd", {})
    conf["collectd"]["configDir"] = os.path.join(run_dir, "collectd")

    path = os.path.join(run_dir, "agent.yaml")
    with open(path, "w") as f:
        f.write(yaml.dump(conf))

    return path


@contextmanager
def run_container(name, **kwargs):
    client = docker.from_env()
    container = client.containers.run(name, detach=True, **kwargs)

    def has_ip_addr():
        container.reload()
        return container.attrs["NetworkSettings"]["IPAddress"]

    wait_for(has_ip_addr, timeout_seconds=5)
    try:
        yield container
    finally:
        print("Container %s logs: %s" % (name, container.logs()))
        container.remove(force=True, v=True)


@contextmanager
def run_service(name):
    client = docker.from_env()
    print(TEST_SERVICES_DIR)
    nginx_image, logs = client.images.build(path=os.path.join(TEST_SERVICES_DIR, name))
    with run_container(nginx_image.id) as cont:
        yield cont


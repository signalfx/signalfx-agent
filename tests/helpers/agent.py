import os
import shutil
import subprocess
import tempfile
from contextlib import contextmanager

import yaml
from tests.paths import AGENT_BIN

from tests.helpers import fake_backend
from tests.helpers.formatting import print_dp_or_event
from tests.helpers.internalmetrics import InternalMetricsClient
from tests.helpers.profiling import PProfClient
from tests.helpers.util import get_unique_localhost, print_lines, retry_on_ebadf, run_subprocess


# pylint: disable=too-many-arguments,too-many-instance-attributes
class Agent:
    def __init__(
        self, run_dir, config, fake_services, config_path=None, debug=True, host=None, env=None, profiling=False
    ):
        assert host is not None
        self.run_dir = run_dir
        self.fake_services = fake_services
        self.debug = debug
        self.pid = None
        self.get_output = None

        self.host = host

        self.env = env
        self.profiling = profiling
        self.config_path = config_path or os.path.join(self.run_dir, "agent.yaml")
        self.config = yaml.safe_load(config)

    def fill_in_config(self):
        run_dir = self.run_dir

        if self.config.get("intervalSeconds") is None:
            self.config["intervalSeconds"] = 3

        if self.config.get("signalFxAccessToken") is None:
            self.config["signalFxAccessToken"] = "testing123"

        if self.fake_services:
            self.config["ingestUrl"] = self.fake_services.ingest_url
            self.config["apiUrl"] = self.fake_services.api_url

        self.config["internalStatusHost"] = self.host
        self.config["internalStatusPort"] = 8095
        if self.profiling:
            self.config["profiling"] = True
            self.config["profilingHost"] = self.host

        self.config["logging"] = dict(level="debug" if self.debug else "info")

        self.config["collectd"] = self.config.get("collectd", {})
        self.config["collectd"]["configDir"] = os.path.join(run_dir, "collectd")
        self.config["collectd"]["logLevel"] = "info"

        self.config["configSources"] = self.config.get("configSources", {})
        self.config["configSources"]["file"] = self.config["configSources"].get("file", {})
        self.config["configSources"]["file"]["pollRateSeconds"] = 1

        self.config["writer"] = self.config.get("writer", {})
        self.config["writer"].setdefault("propertiesSendDelaySeconds", 1)

        if self.config["writer"].get("splunk", {}).get("enabled", False):
            self.config["writer"]["splunk"]["url"] = self.fake_services.splunk_hec_url

    @retry_on_ebadf
    def write_config(self):
        with open(self.config_path, "wb+") as fd:
            print("CONFIG: %s\n%s" % (self.config_path, self.config))
            fd.write(self.get_final_config_yaml().encode("utf-8"))

    def get_final_config_yaml(self):
        self.fill_in_config()
        return yaml.dump(self.config)

    def update_config(self, config_text):
        self.config = yaml.safe_load(config_text)
        self.write_config()

    @property
    def pprof_client(self):
        return PProfClient(self.config["profilingHost"], self.config.get("profilingPort", 6060))

    @property
    def internal_metrics_client(self):
        return InternalMetricsClient(self.config["internalStatusHost"], self.config["internalStatusPort"])

    @property
    def current_status_text(self):
        status_proc = subprocess.run(
            [str(AGENT_BIN), "status", "-config", self.config_path],
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            close_fds=False,
            encoding="utf-8",
            check=False,
        )
        return status_proc.stdout

    @property
    def output(self):
        return self.get_output()

    @contextmanager
    def run_as_subproc(self):
        self.write_config()

        with run_subprocess(
            [AGENT_BIN, "-config", self.config_path] + (["-debug"] if self.debug else []), env=self.env, close_fds=False
        ) as [get_output, pid]:
            self.pid = pid
            self.get_output = get_output
            try:
                yield
            finally:
                print("\nAgent output:")
                print_lines(self.get_output())
                if self.debug:
                    print("\nDatapoints received:")
                    for dp in self.fake_services.datapoints:
                        print_dp_or_event(dp)
                    print("\nEvents received:")
                    for event in self.fake_services.events:
                        print_dp_or_event(event)
                    print("\nTrace spans received:")
                    for span in self.fake_services.spans:
                        print(span)
                    print(f"\nDimensions set: {self.fake_services.dims}")

                    print("\nSplunk log entries received:")
                    for entry in self.fake_services.splunk_entries:
                        print(entry)

    @classmethod
    @contextmanager
    def run(
        cls,
        init_config,
        debug=True,
        fake_services=None,
        backend_options=None,
        host=None,
        extra_env=None,
        profiling=False,
    ):
        with ensure_fake_backend(
            host=host, backend_options=backend_options, fake_services=fake_services
        ) as _fake_services:
            run_dir = tempfile.mkdtemp()
            try:
                agent_env = {**os.environ.copy(), **(extra_env or {})}
                agent = cls(
                    config=init_config,
                    run_dir=run_dir,
                    fake_services=_fake_services,
                    env=agent_env,
                    host=_fake_services.ingest_host,  # This should be unique per test run
                    profiling=profiling,
                    debug=debug,
                )
                with agent.run_as_subproc():
                    yield agent
            finally:
                shutil.rmtree(run_dir, ignore_errors=True)


@contextmanager
def ensure_fake_backend(host=None, backend_options=None, fake_services=None):
    if host is None:
        host = get_unique_localhost()

    if fake_services is None:
        with fake_backend.start(host, **(backend_options or {})) as started_fake_services:
            yield started_fake_services
    else:
        yield fake_services

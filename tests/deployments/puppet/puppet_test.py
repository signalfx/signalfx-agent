import os
import string
import tempfile
from contextlib import contextmanager
from functools import partial as p
from pathlib import Path

import pytest

from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.util import print_lines, wait_for, copy_file_into_container
from tests.packaging.common import (
    INIT_SYSTEMD,
    INIT_UPSTART,
    get_agent_logs,
    is_agent_running_as_non_root,
    run_init_system_image,
)
from tests.paths import REPO_ROOT_DIR

pytestmark = [pytest.mark.puppet, pytest.mark.deployment]

DOCKERFILES_DIR = Path(__file__).parent.joinpath("images").resolve()

APT_MODULE_VERSION = "7.0.0"

DEB_DISTROS = [
    ("debian-8-jessie", INIT_SYSTEMD),
    ("debian-9-stretch", INIT_SYSTEMD),
    ("ubuntu1404", INIT_UPSTART),
    ("ubuntu1604", INIT_SYSTEMD),
    ("ubuntu1804", INIT_SYSTEMD),
]

RPM_DISTROS = [
    ("amazonlinux1", INIT_UPSTART),
    ("amazonlinux2", INIT_SYSTEMD),
    ("centos6", INIT_UPSTART),
    ("centos7", INIT_SYSTEMD),
]

CONFIG = string.Template(
    """
class { signalfx_agent:
    config => {
        signalFxAccessToken => 'testing123',
        ingestUrl => '$ingest_url',
        apiUrl => '$api_url',
        monitors => [
            { type => "host-metadata" },
        ],
    }
}
"""
)


@contextmanager
def run_puppet_agent(base_image, init_system):
    dockerfile = os.path.join(DOCKERFILES_DIR, "Dockerfile.%s" % base_image)
    with run_init_system_image(base_image, path=REPO_ROOT_DIR, dockerfile=dockerfile, with_socat=False) as [
        cont,
        backend,
    ]:
        if (base_image, init_system) in DEB_DISTROS:
            code, output = cont.exec_run(f"puppet module install puppetlabs-apt --version {APT_MODULE_VERSION}")
            assert code == 0, output.decode("utf-8")
            print_lines(output)
        with tempfile.NamedTemporaryFile(mode="w") as fd:
            fd.write(CONFIG.substitute(ingest_url=backend.ingest_url, api_url=backend.api_url))
            fd.flush()
            copy_file_into_container(fd.name, cont, "/root/agent.pp")
        code, output = cont.exec_run("puppet apply /root/agent.pp")
        assert code in (0, 2), output.decode("utf-8")
        print_lines(output)
        try:
            yield cont, backend
        finally:
            print("Agent log:")
            print_lines(get_agent_logs(cont, init_system))


@pytest.mark.parametrize(
    "base_image,init_system",
    [pytest.param(distro, init, marks=pytest.mark.deb) for distro, init in DEB_DISTROS]
    + [pytest.param(distro, init, marks=pytest.mark.rpm) for distro, init in RPM_DISTROS],
)
def test_puppet(base_image, init_system):
    with run_puppet_agent(base_image, init_system) as [cont, backend]:
        assert is_agent_running_as_non_root(cont), "Agent is not running as non-root user"
        assert wait_for(p(has_datapoint_with_dim, backend, "plugin", "host-metadata")), "Datapoints didn't come through"

from contextlib import contextmanager
import docker
from io import BytesIO
import os
import glob
import random
import subprocess
import tarfile

PACKAGING_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../packaging"))
RPM_OUTPUT_DIR = os.path.join(PACKAGING_DIR, "rpm/output/x86_64")
DEB_OUTPUT_DIR = os.path.join(PACKAGING_DIR, "deb/output")
DOCKERFILES_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), "images"))

basic_config = """
monitors:
  - type: collectd/signalfx-metadata
  - type: collectd/cpu
  - type: collectd/uptime
"""

def build_base_image(name):
    client = docker.from_env()
    image, logs = client.images.build(
        path=DOCKERFILES_DIR,
        dockerfile=os.path.join(DOCKERFILES_DIR, "Dockerfile.%s" % name))

    return image.id


def get_deb_package_to_test():
    return get_package_to_test(DEB_OUTPUT_DIR, "deb")


def get_rpm_package_to_test():
    return get_package_to_test(RPM_OUTPUT_DIR, "rpm")


def get_package_to_test(output_dir, extension):
    pkgs = glob.glob(os.path.join(output_dir, "*.%s" % extension))
    if not pkgs:
        raise AssertionError("No .%s files found in %s" % (extension, output_dir))

    if len(pkgs) > 1:
        raise AssertionError("More than one .%s file found in %s" % (extension, output_dir))

    return pkgs[0]


# Run an HTTPS proxy inside the container with socat so that our fake backend
# doesn't have to worry about HTTPS.  The cert file must be trusted by the
# container running the agent.
# This is pretty hacky but docker makes it hard to communicate from a container
# back to the host machine (and we don't want to use the host network stack in
# the container due to init systems).  The idea is to bind mount a shared
# folder from the test host to the container that two socat instances use to
# communicate using a file to make the bytes flow between the HTTPS proxy and
# the fake backend.
@contextmanager
def socat_https_proxy(container, target_host, target_port, source_host):
    cert = "/%s.cert" % source_host
    key = "/%s.key" % source_host

    socat_bin = os.path.abspath(os.path.join(os.path.dirname(__file__), "images/socat"))

    code, output = container.exec_run([
        "socat",
        "-v",
        "OPENSSL-LISTEN:443,cert=%s,key=%s,verify=0,bind=127.0.0.1,fork" % (cert, key),
        "UNIX-CONNECT:/tmp/scratch/%s" % container.id],
        stream=True,
        detach=True)

    proc = subprocess.Popen([
        socat_bin,
        "-v",
        "UNIX-LISTEN:/tmp/scratch/%s,fork" % container.id,
        "TCP4:%s:%d" % (target_host, target_port)],
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT)

    try:
        yield
    finally:
        # The socat instance in the container will die with the container
        proc.kill()

# This is more convoluted that it should be but seems to be the simplest way in
# the face of docker-in-docker environments where volume bind mounting is hard.
def copy_file_into_container(path, container, target_path):
    tario = BytesIO()
    tar = tarfile.TarFile(fileobj=tario, mode='w')

    with open(path, 'rb') as f:
        info = tarfile.TarInfo(name=target_path)
        info.size = os.path.getsize(path)

        tar.addfile(info, f)

    tar.close()
    container.put_archive("/", tario.getvalue())

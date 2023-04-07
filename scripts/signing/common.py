import hashlib
import os
import time

import requests


ARTIFACTORY_URL = "https://splunk.jfrog.io/splunk"
ARTIFACTORY_API_URL = f"{ARTIFACTORY_URL}/api"
DEFAULT_ARTIFACTORY_USERNAME = "signalfx-agent"
DEFAULT_TIMEOUT = 300


def upload_file_to_artifactory(src, dest, user, token):
    print(f"uploading {src} to {dest} ...")

    with open(src, "rb") as fd:
        data = fd.read()
        headers = {"X-Checksum-MD5": hashlib.md5(data).hexdigest()}
        resp = requests.put(dest, auth=(user, token), headers=headers, data=data)

        assert resp.status_code == 201, f"upload failed:\n{resp.reason}\n{resp.text}"

        return resp


def artifactory_file_exists(url, user, token):
    return requests.head(url, auth=(user, token)).status_code == 200


def download_artifactory_file(url, dest, user, token):
    print(f"downloading {url} to {dest} ...")

    resp = requests.get(url, auth=(user, token))

    assert resp.status_code == 200, f"download failed:\n{resp.reason}\n{resp.text}"

    with open(dest, "wb") as fd:
        fd.write(resp.content)


def delete_artifactory_file(url, user, token):
    print(f"deleting {url} ...")

    resp = requests.delete(url, auth=(user, token))

    assert resp.status_code == 204, f"delete failed:\n{resp.reason}\n{resp.text}"


def get_md5_from_artifactory(url, user, token):
    if not artifactory_file_exists(url, user, token):
        return None

    resp = requests.get(url, auth=(user, token))

    assert resp.status_code == 200, f"md5 request failed:\n{resp.reason}\n{resp.text}"

    md5 = resp.json().get("checksums", {}).get("md5", "")

    assert md5, f"md5 not found in response:\n{resp.text}"

    return md5


def wait_for_artifactory_metadata(url, orig_md5, user, token, timeout=DEFAULT_TIMEOUT):
    print(f"waiting for {url} to be updated ...")

    start_time = time.time()
    while True:
        assert (time.time() - start_time) < timeout, f"timed out waiting for {url} to be updated"

        new_md5 = get_md5_from_artifactory(url, user, token)

        if new_md5 and str(orig_md5).lower() != str(new_md5).lower():
            break

        time.sleep(5)


def add_artifactory_args(parser):
    parser.add_argument(
        "--artifactory-user",
        type=str,
        default=os.environ.get("ARTIFACTORY_USERNAME", DEFAULT_ARTIFACTORY_USERNAME),
        metavar="ARTIFACTORY_USERNAME",
        required=False,
        help=f"""
            {ARTIFACTORY_URL} username. Defaults to the ARTIFACTORY_USERNAME env var if set,
            otherwise '{DEFAULT_ARTIFACTORY_USERNAME}'.
        """,
    )
    parser.add_argument(
        "--artifactory-token",
        type=str,
        default=os.environ.get("ARTIFACTORY_TOKEN"),
        metavar="ARTIFACTORY_TOKEN",
        required=False,
        help=f"{ARTIFACTORY_URL} token. Required if the ARTIFACTORY_TOKEN env var is not set.",
    )


def check_artifactory_args(args):
    assert args.artifactory_user, f"{ARTIFACTORY_URL} username not set"
    assert args.artifactory_token, f"{ARTIFACTORY_URL} token not set"

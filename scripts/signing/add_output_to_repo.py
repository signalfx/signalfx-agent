#!/usr/bin/env python3

import argparse
import glob
import os
import sys
import tempfile

from common import (
    ARTIFACTORY_URL,
    ARTIFACTORY_API_URL,
    add_artifactory_args,
    add_signing_args,
    artifactory_file_exists,
    check_artifactory_args,
    check_signing_args,
    get_md5_from_artifactory,
    sign_artifactory_metadata,
    sign_file,
    upload_file_to_artifactory,
    wait_for_artifactory_metadata,
)

ARTIFACTORY_DEB_REPO = "signalfx-agent-deb"
ARTIFACTORY_DEB_REPO_URL = f"{ARTIFACTORY_URL}/{ARTIFACTORY_DEB_REPO}"
ARTIFACTORY_RPM_REPO = "signalfx-agent-rpm-local"
ARTIFACTORY_RPM_REPO_URL = f"{ARTIFACTORY_URL}/{ARTIFACTORY_RPM_REPO}"
DEFAULT_TIMEOUT = 900
PACKAGE_TYPES = ("deb", "rpm")
STAGES = ("test", "beta", "release")


def getargs():
    parser = argparse.ArgumentParser(
        formatter_class=argparse.RawDescriptionHelpFormatter,
        description="Sign and add deb/rpm packages to artifactory.",
    )
    parser.add_argument(
        "path",
        type=str,
        metavar="PATH",
        help="Path to a deb/rpm package file or to a directory containing deb/rpm packages.",
    )
    parser.add_argument(
        "package_type",
        type=str,
        metavar="PACKAGE_TYPE",
        choices=PACKAGE_TYPES,
        help=f"Package type for PATH {PACKAGE_TYPES}.",
    )
    parser.add_argument("stage", type=str, metavar="STAGE", choices=STAGES, help=f"Stage for package(s) {STAGES}.")
    parser.add_argument(
        "--timeout",
        type=int,
        default=DEFAULT_TIMEOUT,
        metavar="TIMEOUT",
        required=False,
        help=f"Signing request timeout in seconds. Defaults to {DEFAULT_TIMEOUT}.",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        default=False,
        required=False,
        help="Never prompt and assume yes when overwriting existing files.",
    )

    add_artifactory_args(parser)
    add_signing_args(parser)

    args = parser.parse_args()

    check_artifactory_args(args)
    if args.stage != "test":
        check_signing_args(args)

    args.path = os.path.abspath(args.path)
    assert os.path.exists(args.path), f"PATH {args.path} not found"

    return args


def add_debs_to_repo(paths, args):
    metadata_api_url = f"{ARTIFACTORY_API_URL}/storage/{ARTIFACTORY_DEB_REPO}/dists/{args.stage}/Release"
    metadata_url = f"{ARTIFACTORY_DEB_REPO_URL}/dists/{args.stage}/Release"

    for path in paths:
        base = os.path.basename(path)
        deb_url = f"{ARTIFACTORY_DEB_REPO_URL}/pool/{base}"
        dest_opts = f"deb.distribution={args.stage};deb.component=main;deb.architecture=amd64;deb.architecture=arm64"
        dest_url = f"{deb_url};{dest_opts}"

        if not args.force and artifactory_file_exists(deb_url, args.artifactory_user, args.artifactory_token):
            overwrite = input(f"package {deb_url} already exists. Overwrite? [y/N] ")
            if overwrite.lower() not in ("y", "yes"):
                sys.exit(1)

        orig_metadata_md5 = get_md5_from_artifactory(metadata_api_url, args.artifactory_user, args.artifactory_token)

        upload_file_to_artifactory(path, dest_url, args.artifactory_user, args.artifactory_token)

        wait_for_artifactory_metadata(
            metadata_api_url, orig_metadata_md5, args.artifactory_user, args.artifactory_token, args.timeout
        )

    return metadata_url


def add_rpms_to_repo(paths, args):
    metadata_api_url = f"{ARTIFACTORY_API_URL}/storage/{ARTIFACTORY_RPM_REPO}/{args.stage}/repodata/repomd.xml"
    metadata_url = f"{ARTIFACTORY_RPM_REPO_URL}/{args.stage}/repodata/repomd.xml"

    for path in paths:
        base = os.path.basename(path)
        dest_url = f"{ARTIFACTORY_RPM_REPO_URL}/{args.stage}/{base}"

        if not args.force and artifactory_file_exists(dest_url, args.artifactory_user, args.artifactory_token):
            overwrite = input(f"package {dest_url} already exists. Overwrite? [y/N] ")
            if overwrite.lower() not in ("y", "yes"):
                sys.exit(1)

        orig_metadata_md5 = get_md5_from_artifactory(metadata_api_url, args.artifactory_user, args.artifactory_token)

        if args.chaperone_token and args.staging_user and args.staging_token:
            with tempfile.TemporaryDirectory() as tmpdir:
                signed_rpm_path = os.path.join(tmpdir, base)
                print(f"Signing {path} (may take 10+ minutes):")
                sign_file(
                    path,
                    signed_rpm_path,
                    "RPM",
                    args.chaperone_token,
                    args.staging_user,
                    args.staging_token,
                    args.timeout,
                )
                upload_file_to_artifactory(signed_rpm_path, dest_url, args.artifactory_user, args.artifactory_token)
        else:
            upload_file_to_artifactory(path, dest_url, args.artifactory_user, args.artifactory_token)

        wait_for_artifactory_metadata(
            metadata_api_url, orig_metadata_md5, args.artifactory_user, args.artifactory_token, args.timeout
        )

    return metadata_url


def main():
    args = getargs()

    if os.path.isdir(args.path):
        paths = glob.glob(f"{args.path}/**/*.{args.package_type}", recursive=True)
    else:
        paths = [args.path]

    if args.package_type == "deb":
        metadata_url = add_debs_to_repo(paths, args)
    else:
        metadata_url = add_rpms_to_repo(paths, args)

    if args.chaperone_token and args.staging_user and args.staging_token:
        sign_artifactory_metadata(
            metadata_url,
            args.artifactory_user,
            args.artifactory_token,
            args.chaperone_token,
            args.staging_user,
            args.staging_token,
            args.timeout,
        )


if __name__ == "__main__":
    main()

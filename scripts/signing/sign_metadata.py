#!/usr/bin/env python3

import argparse

from common import (
    ARTIFACTORY_URL,
    add_artifactory_args,
    add_signing_args,
    check_artifactory_args,
    check_signing_args,
    sign_artifactory_metadata,
)

ARTIFACTORY_DEB_REPO = "signalfx-agent-deb"
ARTIFACTORY_DEB_REPO_URL = f"{ARTIFACTORY_URL}/{ARTIFACTORY_DEB_REPO}"
ARTIFACTORY_RPM_REPO = "signalfx-agent-rpm-local"
ARTIFACTORY_RPM_REPO_URL = f"{ARTIFACTORY_URL}/{ARTIFACTORY_RPM_REPO}"
DEFAULT_TIMEOUT = 300
PACKAGE_TYPES = ("deb", "rpm")
STAGES = ("test", "beta", "release")


def getargs():
    parser = argparse.ArgumentParser(
        formatter_class=argparse.RawDescriptionHelpFormatter,
        description="""
Sign deb/rpm metadata from artifactory.
Should be executed if a package in artifactory is manually deleted/modifed.
""",
    )
    parser.add_argument(
        "package_type",
        type=str,
        metavar="PACKAGE_TYPE",
        choices=PACKAGE_TYPES,
        help=f"Package type for the metadata to sign {PACKAGE_TYPES}.",
    )
    parser.add_argument(
        "stage", type=str, metavar="STAGE", choices=STAGES, help=f"Stage for the metadata to sign {STAGES}."
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=DEFAULT_TIMEOUT,
        metavar="TIMEOUT",
        required=False,
        help=f"Signing request timeout in seconds. Defaults to {DEFAULT_TIMEOUT}.",
    )

    add_artifactory_args(parser)
    add_signing_args(parser)

    args = parser.parse_args()

    check_artifactory_args(args)
    check_signing_args(args)

    return args


def main():
    args = getargs()

    if args.package_type == "deb":
        metadata_url = f"{ARTIFACTORY_DEB_REPO_URL}/dists/{args.stage}/Release"
    else:
        metadata_url = f"{ARTIFACTORY_RPM_REPO_URL}/{args.stage}/repodata/repomd.xml"

    print(f"Signing {metadata_url} (may take a couple of minutes):")

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

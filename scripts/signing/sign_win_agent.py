#!/usr/bin/env python3

import argparse
import os
import tempfile
import zipfile

from common import add_signing_args, check_signing_args, sign_file

BUNDLE_AGENT_PATH = "SignalFxAgent/bin/signalfx-agent.exe"
DEFAULT_TIMEOUT = 1800


def getargs():
    parser = argparse.ArgumentParser(
        formatter_class=argparse.RawDescriptionHelpFormatter, description="Sign the windows agent executable."
    )
    parser.add_argument("path", type=str, metavar="PATH", help="Path to the windows agent bundle zip file.")
    parser.add_argument(
        "--timeout",
        type=int,
        default=DEFAULT_TIMEOUT,
        metavar="TIMEOUT",
        required=False,
        help=f"Signing request timeout in seconds. Defaults to {DEFAULT_TIMEOUT}.",
    )

    add_signing_args(parser)

    args = parser.parse_args()

    check_signing_args(args)

    args.path = os.path.abspath(args.path)
    assert os.path.isfile(args.path), f"PATH {args.path} not found"

    return args


def main():
    args = getargs()

    print(f"Signing {args.path} (may take 20+ minutes):")

    with tempfile.TemporaryDirectory() as tmpdir:
        agent_path = os.path.join(tmpdir, BUNDLE_AGENT_PATH)

        print(f"extracting {args.path} to {tmpdir} ...")
        with zipfile.ZipFile(args.path, mode="r") as bundle:
            filenames = bundle.namelist()
            for filename in bundle.namelist():
                # handle backslashes in bundle
                path = os.path.join(tmpdir, filename.replace("\\", "/"))
                dirname = os.path.join(tmpdir, os.path.dirname(path))
                os.makedirs(dirname, exist_ok=True)
                bundle.extract(filename, path=dirname)
                os.rename(os.path.join(dirname, filename), path)

        sign_file(
            agent_path, agent_path, "WIN", args.chaperone_token, args.staging_user, args.staging_token, args.timeout
        )

        signed_dir = os.path.join(os.path.dirname(args.path), "signed")
        os.makedirs(signed_dir, exist_ok=True)
        new_bundle_path = os.path.join(signed_dir, os.path.basename(args.path))

        print(f"creating {new_bundle_path} with signed agent executable ...")
        with zipfile.ZipFile(new_bundle_path, mode="w", compression=zipfile.ZIP_DEFLATED, compresslevel=1) as bundle:
            os.chdir(tmpdir)
            for root, _, filenames in os.walk("SignalFxAgent"):
                for filename in filenames:
                    bundle.write(os.path.join(root, filename))


if __name__ == "__main__":
    main()

#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
REPO_DIR="$( cd $SCRIPT_DIR/../../ && pwd )"
BUNDLE_PATH=${1-}
if [[ -z "$BUNDLE_PATH" || ! -f "$BUNDLE_PATH" ]]; then
    echo "usage: ./scripts/dependency-check/run.sh BUNDLE_PATH"
    exit 1
fi

BUNDLE_DIR="$( mktemp -d -p /tmp )"
tar -C $BUNDLE_DIR -xf $BUNDLE_PATH

mkdir -p $HOME/.cache/dependency-check
mkdir -p $REPO_DIR/test_output

trap "rm -rf $BUNDLE_DIR" EXIT

docker run --rm \
    -v $BUNDLE_DIR:/bundle \
    -v $HOME/.cache/dependency-check:/usr/share/dependency-check/data \
    -v $REPO_DIR:/src \
    owasp/dependency-check:6.5.0 \
        --scan /bundle \
        --project "$BUNDLE_PATH" \
        --suppression /src/scripts/dependency-check/suppression.xml \
        --out /src/test_output/ \
        --format HTML \
        --failOnCVSS 9 || \
    (echo -e "\nOne or more critical vulnerabilities were found in the agent bundle.\nCheck $REPO_DIR/test_output/dependency-check-report.html, fix the issues, run 'make bundle && make dependency-check', and commit the changes when the issues are resolved." && exit 1)

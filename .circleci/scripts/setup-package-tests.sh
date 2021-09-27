#!/bin/bash

set -eo pipefail

CHANGES_INCLUDE="Dockerfile \
    packaging \
    tests/packaging \
    scripts/patch-interpreter \
    scripts/patch-rpath \
    .circleci/scripts/run-pytest.sh \
    .circleci/config.yml \
    ${BASH_SOURCE[0]}"

if [ "$CIRCLE_BRANCH" != "main" ]; then
    if ! scripts/changes-include-dir $CHANGES_INCLUDE; then
        echo "packaging code has not changed, skipping tests!"
        touch ~/.skip
        exit 0
    fi
fi
export PULL_CACHE=yes
make ${PACKAGE_TYPE}-test-package
# Run non-upgrade tests on node 0, and upgrade tests on node 1
if [ $CIRCLE_NODE_INDEX -eq 0 ]; then
    echo "export MARKERS='$PACKAGE_TYPE and not upgrade'" >> $BASH_ENV
else
    echo "export MARKERS='$PACKAGE_TYPE and upgrade'" >> $BASH_ENV
fi

#!/bin/bash

set -eo pipefail

CHANGES_INCLUDE="deployments/installer \
    tests/packaging/installer_test.py \
    tests/packaging/common.py \
    tests/packaging/images \
    .circleci/scripts/run-pytest.sh \
    .circleci/config.yml \
    ${BASH_SOURCE[0]}"

if [ "$CIRCLE_BRANCH" != "main" ]; then
    if ! scripts/changes-include-dir $CHANGES_INCLUDE; then
        echo "Installer code has not changed, skipping tests!"
        touch ~/.skip
        exit 0
    fi
fi

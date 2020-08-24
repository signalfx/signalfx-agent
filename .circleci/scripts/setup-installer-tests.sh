#!/bin/bash

set -eo pipefail

if [ "$CIRCLE_BRANCH" != "master" ]; then
    if ! scripts/changes-include-dir deployments/installer tests/packaging/installer_test.py tests/packaging/common.py tests/packaging/images .circleci/scripts/run-pytest.sh ${BASH_SOURCE[0]}; then
        echo "Installer code has not changed, skipping tests!"
        touch ~/.skip
        exit 0
    fi
fi

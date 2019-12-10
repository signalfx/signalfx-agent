#!/bin/bash

set -eo pipefail

if [ "$CIRCLE_BRANCH" != "master" ]; then
    if ! scripts/changes-include-dir deployments/installer tests/packaging/installer_test.py tests/packaging/common.py tests/packaging/images ${BASH_SOURCE[0]}; then
        echo "Installer code has not changed, skipping tests!"
        touch ~/.skip
        exit 0
    fi
fi
# Run rpm tests on node 0, and deb tests on node 1
if [ $CIRCLE_NODE_INDEX -eq 0 ]; then
    echo "export MARKERS='installer and rpm'" >> $BASH_ENV
else
    echo "export MARKERS='installer and deb'" >> $BASH_ENV
fi

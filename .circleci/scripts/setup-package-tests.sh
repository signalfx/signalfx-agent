#!/bin/bash

set -eo pipefail

if [ "$CIRCLE_BRANCH" != "master" ]; then
    if ! scripts/changes-include-dir Dockerfile packaging tests/packaging scripts/patch-interpreter scripts/patch-rpath ${BASH_SOURCE[0]}; then
        echo "packaging code has not changed, skipping tests!"
        touch ~/.skip
        exit 0
    fi
fi

if [ "$PACKAGE_TYPE" = "deb" ]; then
    mkdir -p packaging/deb/output/
    mv /tmp/workspace/signalfx-agent*.deb packaging/deb/output/
elif [ "$PACKAGE_TYPE" = "rpm" ]; then
    mkdir -p packaging/rpm/output/x86_64/
    mv /tmp/workspace/signalfx-agent*.rpm packaging/rpm/output/x86_64/
fi

# Run non-upgrade tests on node 0, and upgrade tests on node 1
if [ $CIRCLE_NODE_INDEX -eq 0 ]; then
    echo "export MARKERS='$PACKAGE_TYPE and not upgrade'" >> $BASH_ENV
else
    echo "export MARKERS='$PACKAGE_TYPE and upgrade'" >> $BASH_ENV
fi

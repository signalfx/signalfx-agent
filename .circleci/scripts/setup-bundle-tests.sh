#!/bin/bash

set -eo pipefail

if [ "$CIRCLE_BRANCH" != "master" ]; then
    if ! scripts/changes-include-dir Dockerfile packaging tests/packaging scripts/patch-interpreter scripts/patch-rpath scripts/build tests/requirements.txt ${BASH_SOURCE[0]}; then
        echo "packaging code has not changed, skipping tests!"
        touch ~/.skip
        exit 0
    fi
fi

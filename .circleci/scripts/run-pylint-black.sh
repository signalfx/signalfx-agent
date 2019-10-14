#!/bin/bash

set -eo pipefail

[ -n "$TARGET" ] || (echo "TARGET not defined!" && exit 1)

TARGET_DIR=$TARGET
if [ "$TARGET" = "pytest" ]; then
    TARGET_DIR="tests"
fi
if [ "$CIRCLE_BRANCH" != "master" ]; then
    if ! scripts/changes-include-dir $TARGET_DIR ${BASH_SOURCE[0]}; then
        echo "$TARGET code has not changed, skipping pylint/black!"
        exit 0
    fi
fi
if [ "$TARGET" = "pytest" ]; then
    pip install -q -r tests/requirements.txt
else
    pip install --no-use-pep517 -q -e python
    pip install -q -r python/test-requirements.txt
fi
(make lint-$TARGET && git diff --exit-code $TARGET_DIR ) || \
    (echo "Pylint/black issue(s) found in $TARGET_DIR directory. Run \`make lint-$TARGET\` in the dev image, resolve the issues, and commit the changes." && exit 1)

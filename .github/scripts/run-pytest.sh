#!/bin/bash

set -eo pipefail

[ -f ~/.skip ] && echo "Found ~/.skip, skipping tests!" && exit 0

[ -n "$TESTS_DIR" ] || (echo "TESTS_DIR not defined!" && exit 1)
[ -d "$TESTS_DIR" ] || (echo "Directory '$TESTS_DIR' not found!" && exit 1)

[ -f $GITHUB_ENV ] && source $GITHUB_ENV

unset LD_LIBRARY_PATH

mkdir -p /tmp/scratch
mkdir -p "$HOME/$RESULT_PATH"

TESTS=$TESTS_DIR

PYTEST_PATH="pytest"

sudo sysctl -w vm.max_map_count=262144
REPORT_OPTIONS=""
if [ ${GITHUB_NODE_TOTAL:-0} -gt 0 ] && [ ${GITHUB_NODE_GROUP:-0} -gt 0 ]; then
    REPORT_OPTIONS="--cov --splits $GITHUB_NODE_TOTAL --group $GITHUB_NODE_GROUP"
fi
REPORT_OPTIONS="$REPORT_OPTIONS --verbose --junitxml=$HOME/$RESULT_PATH/results.xml --html=$HOME/$RESULT_PATH/results.html --self-contained-html"

set -x
set +e

$PYTEST_PATH -m "$MARKERS" -n $WORKERS $PYTEST_OPTIONS $REPORT_OPTIONS $TESTS
RC=$?

# re-run failed tests if xdist workers crashed
if [ $RC -ne 0 ] && grep -q 'worker.*crashed' $HOME/$RESULT_PATH/results.html; then
    $PYTEST_PATH -m "$MARKERS" -n $WORKERS $PYTEST_OPTIONS $REPORT_OPTIONS $TESTS
    RC=$?
fi

exit $RC

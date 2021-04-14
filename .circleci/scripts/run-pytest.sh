#!/bin/bash

set -eo pipefail

[ -f ~/.skip ] && echo "Found ~/.skip, skipping tests!" && exit 0

[ -n "$TESTS_DIR" ] || (echo "TESTS_DIR not defined!" && exit 1)
[ -d "$TESTS_DIR" ] || (echo "Directory '$TESTS_DIR' not found!" && exit 1)

[ -f $BASH_ENV ] && source $BASH_ENV
export BUNDLE_DIR=${BUNDLE_DIR:-$(pwd)/bundle}
export AGENT_BIN=${AGENT_BIN:-${BUNDLE_DIR}/bin/signalfx-agent}
export TEST_SERVICES_DIR=${TEST_SERVICES_DIR:-$(pwd)/test-services}

mkdir -p /tmp/scratch
mkdir -p ~/testresults

if [[ $CIRCLE_NODE_TOTAL -gt 1 && -n "$MARKERS" && $SPLIT -eq 1 ]]; then
    # Collect tests based on MARKERS and split them for parallelism
    TESTS=$(python .circleci/scripts/collect_tests.py "$MARKERS" $TESTS_DIR | \
        circleci tests split --split-by=timings --total=$CIRCLE_NODE_TOTAL --index=$CIRCLE_NODE_INDEX)
    [ -n "$TESTS" ] || (echo "No test files found after splitting based on '$MARKERS' marker(s)!" && exit 1)
else
    TESTS=$TESTS_DIR
fi

PYTEST_PATH="pytest"
if [ $WITH_SUDO -eq 1 ]; then
    PYTEST_PATH="sudo -E $PYENV_ROOT/shims/pytest"
fi

sudo sysctl -w vm.max_map_count=262144

REPORT_OPTIONS="--verbose --junitxml=~/testresults/results.xml --html=~/testresults/results.html --self-contained-html"

set -x
set +e

$PYTEST_PATH -m "$MARKERS" -n "$WORKERS" $PYTEST_OPTIONS $REPORT_OPTIONS $TESTS
RC=$?

# re-run failed tests if xdist workers crashed
if [ $RC -ne 0 ] && grep -q 'worker.*crashed' ~/testresults/results.html; then
    $PYTEST_PATH -m "$MARKERS" $PYTEST_OPTIONS $REPORT_OPTIONS $TESTS
    RC=$?
fi

exit $RC

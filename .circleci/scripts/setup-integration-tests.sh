#!/bin/bash

set -eo pipefail

BUNDLE_DIR="$(pwd)/bundle"
AGENT_BIN="$BUNDLE_DIR/bin/signalfx-agent"
TEST_SERVICES_DIR="$(pwd)/test-services"
MARKERS="${MARKERS:-integration}"

mkdir -p "$BUNDLE_DIR"
cid=$(docker create quay.io/signalfx/signalfx-agent-dev:latest true)
docker export $cid | tar -C "$BUNDLE_DIR" -xf -
$BUNDLE_DIR/bin/patch-interpreter $BUNDLE_DIR
docker rm -fv $cid
[ -f "$AGENT_BIN" ] || (echo "$AGENT_BIN not found!" && exit 1)

if [ "$CIRCLE_BRANCH" != "master" ] && ! scripts/changes-include-dir ${BASH_SOURCE[0]}; then
    if ! scripts/changes-include-dir $(find . -iname "*devstack*" -o -iname "*openstack*" | sed 's|^\./||' | grep -v '^docs/'); then
        MARKERS="$MARKERS and not openstack"
    fi
    if ! scripts/changes-include-dir $(find . -iname "*conviva*" | sed 's|^\./||' | grep -v '^docs/'); then
        MARKERS="$MARKERS and not conviva"
    fi
    if ! scripts/changes-include-dir $(find . -iname "*jenkins*" | sed 's|^\./||' | grep -v '^docs/'); then
        MARKERS="$MARKERS and not jenkins"
    fi
fi

echo "export BUNDLE_DIR='$BUNDLE_DIR'" >> $BASH_ENV
echo "export AGENT_BIN='$AGENT_BIN'" >> $BASH_ENV
echo "export TEST_SERVICES_DIR='$TEST_SERVICES_DIR'" >> $BASH_ENV
echo "export MARKERS='$MARKERS'" >> $BASH_ENV
echo "export SPLIT=1" >> $BASH_ENV

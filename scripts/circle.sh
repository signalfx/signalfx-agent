#!/bin/bash

set -e -o pipefail

if [ "$1" = "test" ]; then
    go get github.com/tebeka/go2xunit
    go get github.com/golang/lint/golint
    mkdir -p $CIRCLE_TEST_REPORTS/reports

    cd ../.go_workspace/src/github.com/signalfx/neo-agent
    make lint vet
    go test -v `glide novendor` | go2xunit > $CIRCLE_TEST_REPORTS/reports/unit.xml
fi

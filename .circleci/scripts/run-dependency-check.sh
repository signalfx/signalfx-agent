#!/bin/sh

mkdir -p ./bundle
mkdir -p ./reports
tar -C ./bundle -xf /tmp/workspace/signalfx-agent-latest.tar.gz

/usr/share/dependency-check/bin/dependency-check.sh \
    --scan ./bundle \
    --project "signalfx-agent-latest.tar.gz" \
    --suppression ./scripts/dependency-check/suppression.xml \
    --out ./reports/ \
    --format HTML \
    --format JUNIT \
    --junitFailOnCVSS 9 \
    --failOnCVSS 9 || \
    (echo -e "\nOne or more critical vulnerabilities were found in the agent bundle.\nCheck the report artifact, fix the issues, run 'make bundle && make dependency-check', and commit the changes when the issues are resolved." && exit 1)

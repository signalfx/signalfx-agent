#!/bin/bash

set -eo pipefail

[ -n "$K8S_VERSION" ] || (echo "K8S_VERSION not defined!" && exit 1)

K8S_MIN_VERSION="${K8S_MIN_VERSION:-v1.15.0}"
K8S_MAX_VERSION="${K8S_MAX_VERSION:-v1.19.0}"
K8S_SFX_AGENT="${K8S_SFX_AGENT:-quay.io/signalfx/signalfx-agent-dev:latest}"
WITH_CRIO=${WITH_CRIO:-0}

CHANGES_INCLUDE="deployments/k8s \
    tests/deployments/helm \
    Dockerfile \
    go.mod \
    go.sum \
    .circleci/scripts/run-pytest.sh \
    ${BASH_SOURCE[0]} \
    $(find . -iname '*k8s*' -o -iname '*kube*' | sed 's|^\./||' | grep -v '^docs/')"

if [ "$CIRCLE_BRANCH" != "master" ] && ! scripts/changes-include-dir $CHANGES_INCLUDE; then
    # Only run k8s tests for crio, K8S_MIN_VERSION, and K8S_MAX_VERSION if there are no relevant changes.
    if [[ $WITH_CRIO -ne 1 && "$K8S_VERSION" != "$K8S_MIN_VERSION" && "$K8S_VERSION" != "$K8S_MAX_VERSION" ]]; then
        echo "Skipping kubernetes $K8S_VERSION integration tests."
        touch ~/.skip
        exit 0
    fi
fi

# Push agent image to local registry
[ -f ~/.skip ] && exit 0
docker run -d -e "REGISTRY_HTTP_ADDR=0.0.0.0:5000" -p 5000:5000 registry:2.7
docker run --rm --net=host jwilder/dockerize:0.6.1 -wait tcp://localhost:5000 -timeout 10s
docker tag $K8S_SFX_AGENT localhost:5000/signalfx-agent:latest
docker push localhost:5000/signalfx-agent:latest

#!/bin/bash

set -ex

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

. $SCRIPT_DIR/common.sh

AGENT_IMAGE_NAME="quay.io/signalfuse/signalfx-agent"

if [ -n "${BUILD_BRANCH}" ]; then
  TAG=${TAG:-$BUILD_BRANCH}
else
  TAG=${TAG:-$USER}
fi

do_docker_build ${AGENT_IMAGE_NAME}:${TAG} $SCRIPT_DIR/../Dockerfile

if [ "$BUILD_PUBLISH" = True ]
then
    docker push ${AGENT_IMAGE_NAME}:${TAG}
fi

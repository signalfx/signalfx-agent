#!/bin/bash

set -ex

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

. $SCRIPT_DIR/common.sh

AGENT_IMAGE_NAME=${AGENT_IMAGE_NAME:-"quay.io/signalfuse/signalfx-agent"}

TAG=${AGENT_VERSION:-$($SCRIPT_DIR/current-version)}

do_docker_build ${AGENT_IMAGE_NAME} ${TAG} final-image

if [ "$BUILD_PUBLISH" = True ]
then
    docker push ${AGENT_IMAGE_NAME}:${TAG}
fi

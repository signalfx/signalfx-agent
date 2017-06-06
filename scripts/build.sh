#!/bin/bash

set -ex -o pipefail

. VERSIONS
BUILD_TIME=`date +%FT%T%z`
AGENT_IMAGE_NAME="quay.io/signalfuse/signalfx-agent"
BUILDER_IMAGE_NAME="agent-builder-image"
PROJECT_DIR=${PROJECT_DIR:-${PWD}}
GO_PACKAGES=(
    cmd
    config
    pipelines
    plugins
    secrets
    services
    utils
    watchers
)

# For Jenkins.
if [ -n "${BASE_DIR}" ] && [ -n "${JOB_NAME}" ]; then
    SRC_ROOT=${BASE_DIR}/${JOB_NAME}/neo-agent
else
    SRC_ROOT=${PWD}
fi

if [ -n "${BUILD_BRANCH}" ]; then
    TAG=${BUILD_BRANCH}
else
    TAG=${USER}
fi

BUILD_ROOT=.build
BUILDER_IMAGE_ROOT=${BUILD_ROOT}/builder-image
AGENT_IMAGE_ROOT=${BUILD_ROOT}/agent-image
rm -rf ${BUILD_ROOT}

# Create build image with collectd, Go dependencies, and agent build.
mkdir -p ${BUILDER_IMAGE_ROOT}
cp ${PROJECT_DIR}/scripts/agent-builder-image/Dockerfile ${BUILDER_IMAGE_ROOT}
cp -r ${PROJECT_DIR}/scripts/build-collectd.sh collectd-ext VERSIONS ${BUILDER_IMAGE_ROOT}
rm -rf ${BUILDER_IMAGE_ROOT}/collectd-ext/stub

mkdir -p ${BUILDER_IMAGE_ROOT}/src
cp glide.{yaml,lock} ${BUILDER_IMAGE_ROOT}
cp -r ${GO_PACKAGES[@]} ${BUILDER_IMAGE_ROOT}/src

# Build the builder image.
(cd ${BUILDER_IMAGE_ROOT} && docker build \
    --tag ${BUILDER_IMAGE_NAME}:${TAG} \
    --build-arg collectd_version="${COLLECTD_VERSION}" \
    --build-arg build_time="${BUILD_TIME}" .)

mkdir -p ${AGENT_IMAGE_ROOT}
# Copy collectd and Go agent binaries into the agent-image staging directory.
docker run --rm -v ${SRC_ROOT}/${AGENT_IMAGE_ROOT}:/opt/build ${BUILDER_IMAGE_NAME}:${TAG} \
    bash -c "cp /usr/local/lib/collectd/{libcollectd,java,nginx,python,aggregation}.so /usr/local/lib/collectd/generic-jmx.jar /opt/go/bin/agent /opt/build"

cp ${PROJECT_DIR}/scripts/agent-image/* ${AGENT_IMAGE_ROOT}
cp -r etc ${AGENT_IMAGE_ROOT}

# Build final neo-agent image.
(cd ${AGENT_IMAGE_ROOT} && docker build \
    --build-arg signalfx_agent_version="$SIGNALFX_AGENT_VERSION" \
    --tag ${AGENT_IMAGE_NAME}:${TAG} .)

if [ "$BUILD_PUBLISH" = True ]; then
    docker push ${AGENT_IMAGE_NAME}:${TAG}
fi

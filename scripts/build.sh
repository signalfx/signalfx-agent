#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

v() {
  bash $SCRIPT_DIR/../VERSIONS $1
}

AGENT_IMAGE_NAME="quay.io/signalfuse/signalfx-agent"
TAG=${BUILD_BRANCH:-$USER}

make_go_package_tar() {
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

  # A hack to simplify Dockerfile since Dockerfile doesn't support copying
  # multiple directories without flattening them out
  (cd $SCRIPT_DIR/.. && tar -cf $SCRIPT_DIR/go_packages.tar ${GO_PACKAGES[@]})
}

# If this isn't true then let build use default
if [[ $DEBUG == 'true' ]]
then
  extra_cflags_build_arg="--build-arg extra_cflags='-g -O0'"
fi

make_go_package_tar

docker build \
  --tag ${AGENT_IMAGE_NAME}:${TAG} \
  --label agent.version=$(v SIGNALFX_AGENT_VERSION) \
  --label collectd.version=$(v COLLECTD_VERSION) \
  --build-arg DEBUG=$DEBUG \
  --build-arg collectd_version=$(v COLLECTD_VERSION) \
  --build-arg agent_version=$(v SIGNALFX_AGENT_VERSION) \
  $extra_cflags_build_arg \
  .

if [ "$BUILD_PUBLISH" = True ]
then
    docker push ${AGENT_IMAGE_NAME}:${TAG}
fi

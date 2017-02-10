#!/bin/bash
set -ex -o pipefail

AGENT_IMAGE_BUILD_DIR=`mktemp -d`
BUILD_CONTAINER_ID=""
BUILD_IMAGE_NAME="agent-builder:latest"
SRC_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BUILD_BRANCH=${BUILD_BRANCH:-$USER}

LOCAL_BIN="${AGENT_IMAGE_BUILD_DIR}/scripts/agent-image/bin"
REMOTE_PROJECT="/root/work/neo-agent"
REMOTE_BIN="$REMOTE_PROJECT/.bin"
GO_PACKAGES=(
     'cmd'
     'pipelines'
     'plugins'
     'secrets'
     'services'
     'utils'
   )

trap cleanup_build_containers EXIT

function run_build_container(){
    BUILD_CONTAINER_ID=$(docker run \
    -d \
    $BUILD_IMAGE_NAME tail -f /dev/null)

  echo "$BUILD_CONTAINER_ID"
}

function stop_build_container(){

  if [ "$BUILD_CONTAINER_ID" != "" ]; then
    if docker ps | grep -q "$BUILD_CONTAINER_ID"; then
     docker stop -t 0 $BUILD_CONTAINER_ID
     echo "container $BUILD_CONTAINER_ID stopped"
    else
     echo "container $BUILD_CONTAINER_ID not running (stop_buld_container)"
    fi
  else
    echo "WARN: build container id not set (stop_build_container)"
  fi
}

function remove_build_container(){

  if [ "$BUILD_CONTAINER_ID" != "" ]; then
    if docker ps -a | grep -q "$BUILD_CONTAINER_ID"; then
     docker rm -f $BUILD_CONTAINER_ID
     echo "container $BUILD_CONTAINER_ID removed"
    else
     echo "container $BUILD_CONTAINER_ID not found (remove_build_container)"
    fi
  else
    echo "WARN: container id not set (remove_build_container)"
  fi
}

function cleanup_build_containers(){

  CONTAINERS=$(docker ps -a --filter=ancestor=${BUILD_IMAGE_NAME})
  while read -r line;
  do
    BUILD_CONTAINER_ID=`echo "$line" | awk '{ print $1 }'`
    if [ "$BUILD_CONTAINER_ID" != "CONTAINER" ]; then
      stop_build_container
      remove_build_container
    fi
  done <<< "$CONTAINERS"

  echo "cleanup completed"
}

function setup_build_container(){

  run_build_container

  docker cp ${SRC_ROOT}/scripts/build-components.sh $BUILD_CONTAINER_ID:/tmp/build-components.sh
  docker exec $BUILD_CONTAINER_ID chmod +x /tmp/build-components.sh
  docker exec $BUILD_CONTAINER_ID mkdir -p $REMOTE_PROJECT
  docker cp ${SRC_ROOT}/collectd-ext $BUILD_CONTAINER_ID:$REMOTE_PROJECT/collectd-ext

  for pkg in ${GO_PACKAGES[@]}; do
    docker cp ${SRC_ROOT}/$pkg $BUILD_CONTAINER_ID:$REMOTE_PROJECT/
  done
}

function download_built_artifacts(){

  if [ ! -d "$LOCAL_BIN" ]; then
    mkdir -p $LOCAL_BIN
  fi

  docker cp $BUILD_CONTAINER_ID:${REMOTE_BIN}/libcollectd.so ${LOCAL_BIN}/libcollectd.so
  docker cp $BUILD_CONTAINER_ID:${REMOTE_BIN}/python.so ${LOCAL_BIN}/python.so
  docker cp $BUILD_CONTAINER_ID:${REMOTE_BIN}/signalfx-agent ${LOCAL_BIN}/signalfx-agent
}

function build(){

  echo "starting build"

  # setup build dir
  cp -r ${SRC_ROOT}/scripts $AGENT_IMAGE_BUILD_DIR/
  cp -r ${SRC_ROOT}/etc ${AGENT_IMAGE_BUILD_DIR}/scripts/agent-image/

  # build agent build image
  ${AGENT_IMAGE_BUILD_DIR}/scripts/agent-builder-image/build-image.sh

  # setup container for component builds
  setup_build_container

  # build components
  docker exec $BUILD_CONTAINER_ID bash -c "PROJECT_DIR=${REMOTE_PROJECT} /tmp/build-components.sh"

  # put built artifacts in bin for agent image build
  download_built_artifacts

  # remove build containers
  cleanup_build_containers

  local build_args=(
    -t "${BUILD_BRANCH}"
  )

  if [ "$BUILD_PUBLISH" = "True" ]; then
    build_args+=(--publish)
  fi

  # build the agent image
  ${AGENT_IMAGE_BUILD_DIR}/scripts/agent-image/build-image.sh ${build_args[@]}

  echo "done"
}

build

#!/bin/bash
set -x
set +e

BASE_PACKAGE="github.com/signalfx/neo-agent"
CONTAINER_HOSTNAME="agent-build"
CONTAINER_ID=""
GOPATH="/tmp/collectd/go"
IMAGE_ID=""
IMAGE_TAGNAME="agent-builder:latest"
INCLUDE_DIR="/usr/include/collectd"
LIB_DIR="/usr/lib/collectd"
LOCAL_BIN="./.bin"
REMOTE_PROJECT="/root/work/neo-agent"
REMOTE_BIN="$REMOTE_PROJECT/.bin"
PACKAGES=(
     'cmd'
     'plugins'
     'services'
   )

build_image(){
	
  IMAGE_ID=$(docker images -q ${IMAGE_TAGNAME})
  if [[ "$IMAGE_ID" == "" ]]; then
    IMAGE_ID=$(docker build --tag $IMAGE_TAGNAME .)
  fi
  echo "$IMAGE_ID"
}

run_container(){
    if [[ -z $CONTAINER_ID ]]; then
        CONTAINER_NAME_ARGS=""
    else
        CONTAINER_NAME_ARGS="--name $CONTAINER_ID"
    fi

    CONTAINER_ID=$(docker run \
    -d \
    -h $CONTAINER_HOSTNAME \
    $CONTAINER_NAME_ARGS \
    $IMAGE_TAGNAME tail -f /dev/null)

  echo "$CONTAINER_ID"  
}

stop_container(){

  if [ "$CONTAINER_ID" != "" ]; then
    if [[ $(docker ps | grep "$CONTAINER_ID") ]]; then
     docker stop $CONTAINER_ID
     echo "container $CONTAINER_ID stopped"
    else
     echo "container $CONTAINER_ID not running (stop_container)"
    fi
  else
    echo "WARN: container id not set (stop_container)"
  fi
}

remove_container(){

  if [ "$CONTAINER_ID" != "" ]; then
    if [[ $(docker ps -a | grep "$CONTAINER_ID") ]]; then
     docker rm -f $CONTAINER_ID
     echo "container $CONTAINER_ID removed"
    else
     echo "container $CONTAINER_ID not found (remove_container)"
    fi
  else
    echo "WARN: container id not set (remove_container)"  
  fi
}

cleanup(){

  CONTAINERS=$(docker ps -a --filter=ancestor=${IMAGE_TAGNAME})
  while read -r line;
  do
    CONTAINER_ID=`echo "$line" | awk '{ print $1 }'`
    if [ "$CONTAINER_ID" != "CONTAINER" ]; then
      stop_container
      remove_container
    fi
  done <<< "$CONTAINERS"

  echo "cleanup completed"
}

build(){

  echo "starting build"

  # setup
  build_image

  if [ ! -d "$LOCAL_BIN" ]; then
    mkdir -p $LOCAL_BIN
  fi

  run_container

  BUILD_COLLECTD_LIB=true
  BUILD_AGENT=true

  $(docker cp ./build-components.sh $CONTAINER_ID:/tmp/build-components.sh)
  $(docker exec $CONTAINER_ID chmod +x /tmp/build-components.sh)
  $(docker exec $CONTAINER_ID mkdir -p ${REMOTE_PROJECT})

  if [ "$BUILD_COLLECTD_LIB" = true ]; then
    $(docker exec $CONTAINER_ID mkdir -p ${LIB_DIR} ${INCLUDE_DIR})
    $(docker cp ./collectd-ext $CONTAINER_ID:${REMOTE_PROJECT}/collectd-ext)
  fi

  if [ "$BUILD_AGENT" = true ]; then
    for pkg in ${PACKAGES[@]}; do
      $(docker cp ./$pkg $CONTAINER_ID:${REMOTE_PROJECT}/)
    done
  fi

  # build
  echo "building components"
  $(docker exec $CONTAINER_ID /tmp/build-components.sh)
  echo "finished building components"

  # download built artifacts
  if [ "$BUILD_COLLECTD_LIB" = true ]; then
    echo "downloading built collectd library" 
    $(docker cp $CONTAINER_ID:${REMOTE_BIN}/libcollectd.so ${LOCAL_BIN}/libcollectd.so)
    $(docker cp $CONTAINER_ID:${REMOTE_BIN}/python.so ${LOCAL_BIN}/python.so)
  fi

  if [ "$BUILD_AGENT" = true ]; then
    echo "downloading built signalfx-agent"
    $(docker cp $CONTAINER_ID:${REMOTE_BIN}/signalfx-agent ${LOCAL_BIN}/signalfx-agent)
  fi

  # cleanup
  if [ -f "${LOCAL_BIN}/signalfx-agent" ]; then
    cleanup
    echo "build completed!"
  else
    echo "build failed!"
  fi
}

build

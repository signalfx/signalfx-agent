#!/bin/bash
set -xe -o pipefail

BUILD_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
DOCKERFILE="Dockerfile"
IMAGE_ID=""
IMAGE_REGISTRYHOST="quay.io"
IMAGE_REPOSITORY="signalfuse/signalfx-agent"
IMAGE_TAGNAME="dev-latest"
SIGNALFX_AGENT_BIN="bin"
SIGNALFX_AGENT_CONFIGURATIONS="etc"
SIGNALFX_AGENT_VERSION="dev-latest"
BUILD_PUBLISH=0

function build_image(){
  IMAGE_NAME="${IMAGE_REGISTRYHOST}/${IMAGE_REPOSITORY}:${IMAGE_TAGNAME}"
  IMAGE_ID=$(docker images -q $IMAGE_NAME)
  if [ "$IMAGE_ID" != "" ]; then
    remove_image
  fi

  cd $BUILD_DIR
  docker build -f $DOCKERFILE \
    --build-arg signalfx_agent_bin="$SIGNALFX_AGENT_BIN" \
    --build-arg signalfx_agent_configurations="$SIGNALFX_AGENT_CONFIGURATIONS" \
    --build-arg signalfx_agent_version="$SIGNALFX_AGENT_VERSION" \
    --tag ${IMAGE_NAME} .
  if [ "$BUILD_PUBLISH" = 1 ]; then
    docker push ${IMAGE_NAME}
  fi
}

function remove_image(){
  if [ "$IMAGE_ID" != "" ]; then
    if docker images | grep -q "$IMAGE_ID"; then
     docker rmi $IMAGE_ID
     echo "image $IMAGE_ID removed"
    else
     echo "image $IMAGE_ID not found (remove_image)"
    fi
  else
    echo "WARN: image id not set (remove_image)"
  fi
}

while [[  "$#" -gt "0" ]]
do
  key="$1"
  case $key in
    -b|--agent-bin-dir)
      shift
      SIGNALFX_AGENT_BIN="$1"
      ;;
    -c|--agent-configs-dir)
      shift
      SIGNALFX_AGENT_CONFIGURATIONS="$1"
      ;;
    -d|--build-dir)
      shift
      BUILD_DIR="$1"
      ;;
    -f|--dockerfile)
      shift
      DOCKERFILE="$1"
      ;;
    -t|--image-tagname)
      shift
      IMAGE_TAGNAME="$1"
      ;;
    -v|--agent-version)
      shift
      SIGNALFX_AGENT_VERSION="$1"
      ;;
    -p|--publish)
      BUILD_PUBLISH=1
      ;;
    *)
      echo "Unknown Option"
      exit 1
      ;;
  esac
  shift
done

build_image

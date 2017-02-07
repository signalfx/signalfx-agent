#!/bin/bash
set -xe -o pipefail

BUILD_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
DOCKERFILE="Dockerfile"
IMAGE_ID=""
IMAGE_NAME="agent-builder:latest"

function build_image(){
  IMAGE_ID=$(docker images -q $IMAGE_NAME)
  if [ "$IMAGE_ID" == "" ]; then
    cd $BUILD_DIR
    docker build -f $DOCKERFILE --tag ${IMAGE_NAME} .
  fi
}

while [[  "$#" -gt "0" ]]
do
  key="$1"
  case $key in
    -d|--build-dir)
      shift
      BUILD_DIR="$1"
      ;;
    -f|--dockerfile)
      shift
      DOCKERFILE="$1"
      ;;
    -n|--image-name)
      shift
      IMAGE_NAME="$1"
      ;;
    *)
      echo "Unknown Option"
      exit 1
      ;;
  esac
  shift
done

build_image

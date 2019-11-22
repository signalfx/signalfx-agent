#!/bin/bash

set -eo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
IMAGES=$(cat $SCRIPT_DIR/docker-image-cache.txt)
CACHE_DIR="/tmp/docker-cache"

mkdir -p $CACHE_DIR

for image in $IMAGES; do
    tar_path="${CACHE_DIR}/$(echo ${image}.tar | tr ':/' '-')"
    echo "Saving $image as $tar_path"
    docker pull $image
    docker save -o "$tar_path" $image
done

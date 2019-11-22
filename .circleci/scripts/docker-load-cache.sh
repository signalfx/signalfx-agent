#!/bin/bash

set -eo pipefail

CACHE_DIR="/tmp/docker-cache"

mkdir -p $CACHE_DIR

for image in $(find $CACHE_DIR -name "*.tar"); do
    docker load -i $image
done

#!/bin/bash

if [ $# -lt 2 ]; then
    echo "usage: extract-image.sh IMAGE_NAME DIR"
    exit 1
fi

set -euo pipefail

IMAGE_NAME="$1"
BUNDLE_DIR="$2"

[ -d "$BUNDLE_DIR" ] && rm -rf "$BUNDLE_DIR"
mkdir -p "$BUNDLE_DIR"

cid=$( docker create $IMAGE_NAME true )
docker export $cid | tar -C "$BUNDLE_DIR" -xf -
docker rm -fv $cid

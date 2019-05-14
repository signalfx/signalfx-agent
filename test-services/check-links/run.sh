#!/bin/bash

REPO_ROOT_DIR=${1:-"/usr/src/signalfx-agent"}

if [ ! -d "$REPO_ROOT_DIR" ]; then
    echo "$REPO_ROOT_DIR not found!" >&2
    exit 1
fi

if [ ! -f "$REPO_ROOT_DIR/test-services/check-links/config.json" ]; then
    echo "$REPO_ROOT_DIR/test-services/check-links/config.json not found!" >&2
    exit 1
fi

nfailed=0
for f in $(find "$REPO_ROOT_DIR" -name "*.md" -not -path "$REPO_ROOT_DIR/vendor/*" | sort); do
    markdown-link-check -c "$REPO_ROOT_DIR/test-services/check-links/config.json" -qv $f
    (( nfailed += $? ))
done

if [ $nfailed -ne 0 ]; then
    echo -e "\nERROR: dead links found! See output above for details." >&2
    exit 1
fi

exit 0

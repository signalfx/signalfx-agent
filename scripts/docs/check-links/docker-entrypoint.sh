#!/bin/bash

AGENT_DIR="/usr/src/signalfx-agent"
CONFIG="$AGENT_DIR/scripts/docs/check-links/config.json"

if [ $# -eq 0 ]; then
    if [ -z "$(ls -A $AGENT_DIR)" ]; then
        echo "$AGENT_DIR is empty" >&2
        echo "Make sure to mount the signalfx-agent repo directory to $AGENT_DIR" >&2
        exit 1
    fi
    nfailed=0
    for f in $(find "$AGENT_DIR" -name "*.md" -not -path "$AGENT_DIR/vendor/*" | sort); do
        markdown-link-check -c "$CONFIG" -qv "$f"
        (( nfailed += $? ))
    done
    if [ $nfailed -gt 0 ]; then
        echo -e "\nERROR: dead links found! See output above for details." >&2
        exit 1
    fi
else
    exec "$@"
fi

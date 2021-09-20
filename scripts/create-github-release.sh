#!/bin/bash

set -euxo pipefail

tag="$1"
digest="$2"
assets_dir="$3"
image="quay.io/signalfx/signalfx-agent:${tag#v}"

tmpfile=$(mktemp)
cat <<EOH > $tmpfile
⚠️The SignalFx Smart Agent is deprecated. For details, see the [Deprecation Notice](https://github.com/signalfx/signalfx-agent/blob/main/docs/smartagent-deprecation-notice.md) ⚠️

$(git tag -l --format='%(contents:body)' $tag)

> Docker Image: \`$image\` (digest: \`$digest\`)
EOH

cat $tmpfile

gh release create -R https://github.com/signalfx/signalfx-agent -F "$tmpfile" -t "$tag" "$tag" "${assets_dir}"/*

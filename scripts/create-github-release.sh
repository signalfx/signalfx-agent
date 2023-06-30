#!/bin/bash

set -euxo pipefail

tag="$1"
digest="$2"
assets_dir="$3"
image="quay.io/signalfx/signalfx-agent:${tag#v}"

tmpfile=$(mktemp)
cat <<EOH > $tmpfile
⚠️The SignalFx Smart Agent has reached End Of Support ⚠️

> The [Splunk Distribution of OpenTelemetry Collector](https://github.com/signalfx/splunk-otel-collector) is the successor.
>
> To learn how to migrate, see [Migrate from SignalFx Smart Agent to the Splunk Distribution of OpenTelemetry Collector](https://docs.splunk.com/Observability/gdi/opentelemetry/smart-agent-migration-to-otel-collector.html).
>
> Note that this affects the standalone agent; the Smart Agent monitors will be available and supported with the [Smart Agent receiver](https://github.com/signalfx/splunk-otel-collector/blob/main/pkg/receiver/smartagentreceiver/README.md) in the Splunk Distribution of OpenTelemetry Collector.

$(git tag -l --format='%(contents:body)' $tag)

> Docker Image: \`$image\` (digest: \`$digest\`)
EOH

cat $tmpfile

gh release create -R https://github.com/signalfx/signalfx-agent -F "$tmpfile" -t "$tag" "$tag" "${assets_dir}"/*

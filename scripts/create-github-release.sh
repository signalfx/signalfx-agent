#!/bin/bash

set -euxo pipefail

tag="$1"
digest="$2"
assets_dir="$3"
image="quay.io/signalfx/signalfx-agent:${tag#v}"

tmpfile=$(mktemp)
cat <<EOH > $tmpfile
⚠️The SignalFx Smart Agent has reached End Of Support ⚠️

> The [Splunk Distribution of OpenTelemetry Collector](https://docs.splunk.com/Observability/gdi/opentelemetry/opentelemetry.html) is the successor. Smart Agent monitors are available and supported through the [Smart Agent receiver](https://docs.splunk.com/Observability/gdi/opentelemetry/components/smartagent-receiver.html) in the Splunk Distribution of OpenTelemetry Collector.
>
> To learn how to migrate, see [Migrate from SignalFx Smart Agent to the Splunk Distribution of OpenTelemetry Collector](https://docs.splunk.com/Observability/gdi/opentelemetry/smart-agent-migration-to-otel-collector.html).

$(git tag -l --format='%(contents:body)' $tag)

> Docker Image: \`$image\` (digest: \`$digest\`)
EOH

cat $tmpfile

gh release create -R https://github.com/signalfx/signalfx-agent -F "$tmpfile" -t "$tag" "$tag" "${assets_dir}"/*

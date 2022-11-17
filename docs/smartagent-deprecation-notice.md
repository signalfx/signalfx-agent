> # :warning: Deprecation Notice for SignalFx Smart Agent
> **The SignalFx Smart Agent is deprecated and will reach End of Support on June 30th, 2023. After that date, this repository will be archived and no longer receive updates. Until then, only critical security fixes and bug fixes will be provided.**
>
>Going forward, you should use the [Splunk Distribution of OpenTelemetry Collector](https://docs.splunk.com/Observability/gdi/opentelemetry/resources.html), which fully supports the OpenTelemetry standard and includes customizations for:
>* Splunk products
>* Fluentd for log capture
>* Tools to support migration from SignalFx products

>Note that the date affects the standalone agent; the Smart Agent monitors will be available and supported with the [Smart Agent receiver](https://github.com/signalfx/splunk-otel-collector/blob/main/pkg/receiver/smartagent/README.md) in the Splunk Distribution of OpenTelemetry Collector.

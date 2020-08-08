# Deploy the SignalFx Smart Agent and OpenTelemetry Collector for Infrastructure Monitoring

Introduction text here.

Two-step process:

1. Deploy the SignalFx Smart Agent
2. Deploy the OpenTelemetry Collector for SignalFx

## Deploy the SignalFx Smart Agent

The SignalFx platform is built to ingest, store, analyze, visualize, and alert on metrics at scale. A metric is anything that is measurable (you can assign a numerical value to it) and variable (changes over time). The following are examples of metrics:

* CPU utilization % of a server
* Response time in milliseconds of an API call
* The number of unique users who logged in over the previous 24-hour period

You can use the SignalFx Smart Agent to collect metrics for SignalFx Infrastructure Monitoring. See [Use the Smart Agent](https://docs.signalfx.com/en/latest/integrations/agent/index.html#smart-agent "Use the Smart Agent") for more information on using the Smart Agent.

**Step 1: Install the SignalFx Smart Agent**

There are two options for installing the SignalFx Smart Agent:
- Use the [quick installation](https://docs.signalfx.com/en/latest/integrations/agent/quick-install.html "Quick Install") option for simplified SignalFx Smart Agent command-line installation on a single host.
- Use the [advanced installation](https://docs.signalfx.com/en/latest/integrations/agent/advanced-install-options.html "Advanced Installation Options") option for bulk deployments and for configuring various monitors for your environment.

**Step 2: Configure the SignalFx Smart Agent**

The SignalFx Smart Agent is configured primarily by a YAML document located at `/etc/signalfx/agent.yaml`. The location of the configuration file can be specified by the `-config` flag to the agent binary (`signalfx-agent`).

See [Agent Configuration](https://docs.signalfx.com/en/latest/integrations/agent/config-schema.html "Agent Configuration") for complete details, including configuration options and an example configuration file.

## Deploy the OpenTelemetry Collector for SignalFx

Introduction here

### How the OpenTelemetry Collector works

After installing the SignalFx Smart Agent on each host, you can optionally deploy the OpenTelemetry Collector in each datacenter/region/cluster where traced applications run. In general, the OpenTelemetry Collector should receive data from the SignalFx Smart Agent.

The OpenTelemetry Collector uses pipelines to receive, process, and export trace data with components conveniently known as receivers, processors, and exporters. Set up pipelines with services. You can also add extensions that provide an OpenTelemetry Collector with additional functionality, such as diagnostics and health checks. The OpenTelemetry Collector has two versions: a [core version](https://github.com/open-telemetry/opentelemetry-collector> "Core Version") and a [contributions version](https://github.com/open-telemetry/opentelemetry-collector-contrib "Contributions"). The core version provides receivers, processors, and exporters for general use. The contributions version provides receivers, processors, and exporters for specific vendors and use cases.

SignalFx uses the contributions versions described in the following table for receivers and exporters to send data to an OpenTelemetry Collector and to receive data from an OpenTelemetry Collector:

| Component   |     Name      |  Description |
|----------|:-------------:|------:|
| Receiver |  [signal-fx](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/master/receiver/signalfxreceiver) | A receiver is how data gets into the OpenTelemetry Collector. One or more receivers must be configured. By default, no receivers are configured. |
| Processors |    [Various](https://github.com/open-telemetry/opentelemetry-collector/blob/master/processor/README.md)   |   Processors are run on data between being received and being exported.  |
| Exporters |  [signal-fx](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/master/receiver/signalfxreceiver) |   An exporter is how data gets sent to different systems/back-ends. Generally, an exporter translates the internal format into another defined format. |

### Deploy the OpenTelemetry Collector

Procedure and collector.yaml in this section.

# Deploy the SignalFx Smart Agent and OpenTelemetry Collector for Infrastructure Monitoring

SignalFx Infrastructure Monitoring helps you gather metrics related to your system's performance. Metrics provide information about processes running inside the system, including counters, cumulative counters, and gauges.

The OpenTelemetry Collector is a stand-alone service that can ingest metrics from various sources. The OpenTelemetry collector provides additional flexibility in configuration options. You can now configure integrations to send data to an OpenTelemetry Collector to centrally manage data sent to SignalFx.

This is two-step process:

1. [Deploy the SignalFx Smart Agent](#deploy-the-signalfx-smart-agent)
2. [Deploy an OpenTelemetry Collector for SignalFx](#deploy-an-opentelemetry-collector-for-signalfx)

The following sections describe each step in detail.

## Deploy the SignalFx Smart Agent

The SignalFx platform is built to ingest, store, analyze, visualize, and alert on metrics at scale. A metric is anything that is measurable (you can assign a numerical value to it) and variable (changes over time). The following are examples of metrics:

* CPU utilization % of a server
* Response time in milliseconds of an API call
* The number of unique users who logged in over the previous 24-hour period

You can use the SignalFx Smart Agent to collect metrics for SignalFx Infrastructure Monitoring. See [Use the Smart Agent](https://docs.signalfx.com/en/latest/integrations/agent/index.html#smart-agent "Use the Smart Agent") for more information.

**Step 1: Install the SignalFx Smart Agent**

There are two options for installing the SignalFx Smart Agent:
- Use the [quick installation](https://docs.signalfx.com/en/latest/integrations/agent/quick-install.html "Quick Install") option for simplified SignalFx Smart Agent command-line installation on a single host.
- Use the [advanced installation](https://docs.signalfx.com/en/latest/integrations/agent/advanced-install-options.html "Advanced Installation Options") option for bulk deployments and for configuring various monitors for your environment.

**Step 2: Configure the SignalFx Smart Agent**

The SignalFx Smart Agent is configured primarily by a YAML document located at `/etc/signalfx/agent.yaml`. The location of the configuration file can be specified by the `-config` flag to the agent binary (`signalfx-agent`).

See [Agent Configuration](https://docs.signalfx.com/en/latest/integrations/agent/config-schema.html "Agent Configuration") for complete details, including configuration options and an example configuration file.

## Deploy an OpenTelemetry Collector for SignalFx

After installing the SignalFx Smart Agent on each host, you can deploy an OpenTelemetry Collector in each datacenter/region/cluster where traced applications run. In general, an OpenTelemetry Collector should receive data from the SignalFx Smart Agent.

The OpenTelemetry Collector uses pipelines to receive, process, and export trace data with components conveniently known as receivers, processors, and exporters. Set up pipelines with services. You can also add extensions that provide an OpenTelemetry Collector with additional functionality, such as diagnostics and health checks. The OpenTelemetry Collector has two versions: a [core version](https://github.com/open-telemetry/opentelemetry-collector "Core Version") and a [contributions version](https://github.com/open-telemetry/opentelemetry-collector-contrib "Contributions"). The core version provides receivers, processors, and exporters for general use. The contributions version provides receivers, processors, and exporters for specific vendors and use cases.

SignalFx uses the components described in the following table to send data to an OpenTelemetry Collector and to receive data from an OpenTelemetry Collector:

| **Component** | **Name**        | **Description**                                                                                       |  
|---------------| :---------------:  |-------------------------------------------------------------------------------------------------------|
| Receiver      | `signal-fx`     | Component that sets the endpoint for receiving metrics data with the SignalFx metric data format.     |      
| Processor     | Various         | Component that pre-processes data before it is exported.                                              |      
| Exporter      | `signal-fx`     | Component that forwards data to SignalFx with the metric data format.                                 |

  
See [Configuration](https://opentelemetry.io/docs/collector/configuration/ "OpenTelemetry Collector Configuration") for more information about the OpenTelemetry Collector components.

### Deploy an OpenTelemetry Collector

To deploy an OpenTelemetry Collector:

1. Download the [latest release](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases "OpenTelemetry Collector contributions releases") of the OpenTelemetry Collector contributions version from GitHub.
2. Create a configuration file that defines components for the OpenTelemetry Collector. A sample configuration file is provided below.
```
# Save this file as collector.yaml.
# Extensions are optional and are provided to monitor
# the health of the OpenTelemetry Collector.  
    extensions:
      health_check: {}
      pprof: {}
      zpages: {}
# The following section is used to collect the
# OpenTelemetry Collector metrics.
    receivers:
      signalfx:
# The SignalFx receiver accepts metrics in the SignalFx
# proto format. This allows the collector to receive
# metrics from the OpenTelemetry Collector.
    processors:
# Processors are not enabled by default; however, the following
# processors are recommended.
    batch:
# The batch processor must be defined in the pipeline after the
# memory_limiter as well as any sampling processors.
    memory_limiter:
      ballast_size_mib: 683
      check_interval: 2s
      limit_mib: 1800
      spike_limit_mib: 500
 # Enabling the memory_limiter is strongly recommended for every
 # pipeline. Configuration is based on the amount of memory
 # allocated to the collector. The configuration below assumes 2 GB of
 # memory. In general, the ballast should be set to 1/3 of the collector's
 # memory. The limit should be 90% of the collector's memory up to 2 GB. The
 # spike should be 25% of the collector's memory up to 2 GB. In addition,
 # the "--mem-ballast-size-mib" CLI flag must be set to the same value as
 # the "ballast_size_mib".
    exporters:
# One or more exporters must be configured.
# Metrics
     signalfx:
        access_token: "YOUR_ACCESS_TOKEN"
        realm: "YOUR_SIGNALFX_REALM"
     service:
       pipelines:
         metrics:
           receivers: [signalfx]
           processors: [memory_limiter, batch]
           exporters: [signalfx]
       extensions: [health_check, zpages, pprof]
```
3. Deploy the Open Telemetry Collector.
```
otelcontribcol --config collector.yaml --mem-ballast-size-mib=683
```

# Deploy the SignalFx Smart Agent and OpenTelemetry Collector for SignalFx Infrastructure Monitoring

SignalFx Infrastructure Monitoring helps you gather metrics related to your system's performance. Metrics provide information about processes running inside the system, including counters, cumulative counters, and gauges.

OpenTelemetry provides the libraries, agents, and other components that you need to capture telemetry from your services so that you can better observe, manage, and debug them. The following are just a few scenarios where you can use OpenTelemetry to monitor your solutions:

* Creating custom async libraries 
* Instrumenting metrics
* Using multiple context propagation formats (B3, W3C) in parallel

The OpenTelemetry Collector is a stand-alone service that can ingest metrics from various sources. The OpenTelemetry collector provides additional flexibility in configuration options. You can now configure integrations to send data to an OpenTelemetry Collector to centrally manage data sent to SignalFx.

Deploying the SignalFx Smart Agent and OpenTelemetry Collector is a two-step process:

1. [Deploy the SignalFx Smart Agent](#deploy-the-signalfx-smart-agent)
2. [Deploy an OpenTelemetry Collector for SignalFx](#deploy-an-opentelemetry-collector-for-signalfx)

The following sections describe each step in detail.

## Deploy the SignalFx Smart Agent

The SignalFx Smart Agent receives metrics from [integrations](https://docs.signalfx.com/en/latest/integrations/integrations-reference/index.html “Integrations reference guide”) before exporting them to the OpenTelemetry Collector for central management and processing. Monitors are one of the main SignalFX Smart Agent components. Monitors gather metrics from the host and from running applications. See [monitor configuration](https://docs.signalfx.com/en/latest/integrations/agent/monitors/_monitor-config.html “Monitor Configuration”) for configuration options.  

**Step 1: Install the SignalFx Smart Agent**

There are two options for installing the SignalFx Smart Agent:
- Use the [quick installation](https://docs.signalfx.com/en/latest/integrations/agent/quick-install.html "Quick Install") option for simplified SignalFx Smart Agent command-line installation on a single host.
- Use the [advanced installation](https://docs.signalfx.com/en/latest/integrations/agent/advanced-install-options.html "Advanced Installation Options") option for bulk deployments and for configuring various monitors for your environment.

**Step 2: Configure the SignalFx Smart Agent**

The SignalFx Smart Agent is configured primarily by a YAML document located at `/etc/signalfx/agent.yaml`. The location of the configuration file can be specified by the `-config` flag to the agent binary (`signalfx-agent`).

See [Agent Configuration](https://docs.signalfx.com/en/latest/integrations/agent/config-schema.html "Agent Configuration") for configuration options and an example configuration file. See [deployments](https://github.com/signalfx/signalfx-agent/tree/master/deployments "Deployments") for more information on the various ways to deploy the SignalFx Smart Agent.

## Deploy an OpenTelemetry Collector for SignalFx

After installing the SignalFx Smart Agent on each host, you can deploy an OpenTelemetry Collector in each datacenter/region/cluster where traced applications run. In general, an OpenTelemetry Collector should receive data from the SignalFx Smart Agent.

The OpenTelemetry Collector uses pipelines to receive, process, and export trace data with components conveniently known as receivers, processors, and exporters. Set up pipelines with services. You can also add extensions that provide an OpenTelemetry Collector with additional functionality, such as diagnostics and health checks. The OpenTelemetry Collector has two versions: a [core version](https://github.com/open-telemetry/opentelemetry-collector "Core Version") and a [contributions version](https://github.com/open-telemetry/opentelemetry-collector-contrib "Contributions"). The core version provides receivers, processors, and exporters for general use. The contributions version provides receivers, processors, and exporters for specific vendors and use cases.

SignalFx uses the components described in the following table to send data to an OpenTelemetry Collector and to receive data from an OpenTelemetry Collector:

| **Component** | **Name**        | **Description**                                                                                       |  
|---------------| :---------------:  |-------------------------------------------------------------------------------------------------------|
| Receiver      | `signal-fx`     | Component that sets the endpoint for receiving metrics data with the SignalFx metric data format.     |      
| Processor     | `memory_limiter`, `batch`, and `queued_retry` are the recommended processors.         | Component that pre-processes data before it is exported.                                              |      
| Exporter      | `signal-fx`     | Component that forwards data to SignalFx with the metric data format.                                 |

  
See [Configuration](https://opentelemetry.io/docs/collector/configuration/ "OpenTelemetry Collector Configuration") for more information about the OpenTelemetry Collector components.

To deploy an OpenTelemetry Collector:

1. Download the [latest release](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases "OpenTelemetry Collector contributions releases") of the OpenTelemetry Collector contributions version from GitHub.
2. Create a configuration file that defines components for the OpenTelemetry Collector. A sample configuration file is provided below. See [examples](https://github.com/open-telemetry/opentelemetry-collector/tree/master/examples “OTeL examples”) for an OpenTelemetry Collector demo.
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
 # Enabling the memory_limiter is strongly recommended for every pipeline. 
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

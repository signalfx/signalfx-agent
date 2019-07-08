#monitor_template in github

Headings and associated links can be deleted if you are sure they are not used.



_INSTALLATION TAB_



\<monitor logo>
# \<Monitor name>

- [Requirements and Dependancies](#requirements-and-dependancies)
- [Installation](#installation)
- [Configuration](#configuration)

## Requirements and Dependencies

in table format if possible

## Installation

Steps to install this monitor are described below.

__Step 1.__ 

__Step 2.__ 

__Step 3.__ 

## Configuration

The configuration options are described below with sample code snippets.

| Config option | Required | Type                 | Description                  |
| ------------- | -------- | -------------------- | ---------------------------- |
| `kubeletAPI`  | no       | `object (see below)` | Kubelet client configuration |

```sh
snippets
```





_FEATURES TAB_



# Monitor Features

- [Description](#description)
- [Dashboards](#dashboards)
- [Infrastructure Navigator Views](#infrastructure-navigator-views)

## Description

Monitors \<what> using the information provided by \<what> which collects metrics from \<what> instances by hitting these endpoints: 
* <link to item>
* <link to item>


For more information on the source data, see <https://...>

Monitor type: \<type name>

Monitor Source Code \<link>

Accepts Endpoints: <yes/no>

Multiple Instances Allowed: <yes/no>


## Dashboards

The following are examples of built-in dashboards that can be used with your monitor. 



## Infrastructure Navigator Views

The following are built-in navigator views appropriate for this monitor.





_METRICS TAB_



# Metrics

- [Metrics](#metrics)

- [Custom metric configuration](#custom-metric-configuration)

- [Dimensions](#dimensions)

## Metrics

In addition to the common default metrics that are described [here](https://docs.signalfx.com/en/latest/integrations/agent/monitor-config.html), the following table shows additional optional metrics available for this monitor.

- Metrics that are marked as Included in the table below are sent by default by the Smart Agent as part of a host-based subscription, and you are not charged for them.

- Metrics that are not marked as Included are custom metrics, such as system or service metrics that you configure the Smart Agent to send outside of the default set of metrics. Your SignalFx subscription allows you to send a certain number of custom metrics.

You may need to add a flag to these metrics. Check the configuration file for comments about flag requirements.


| Name | Type | Included | Description |
| ---  | ---  | ---    | ---         |
| `name1` | counter | ✔ | Total connections count per broker |
| `name2` | gauge | ✔ | Total number of consumers subscribed to destinations on the broker |
| `name3` | gauge |  | Total number of messages that have been acknowledged from the broker. |


### Custom metric configuration

To collect custom metrics, you must configure your monitor to listen for those metrics and then send those metrics to the agent.

To specify custom metrics, add a _metricsToInclude_ filter to the agent configuration file, as shown in the code snippet below. The sample snippet lists all available custom metrics. Copy and paste the snippet into your monitor configuration file, then delete any custom metrics that you do not want.

Note that some of the custom metrics require you to set a flag in addition to adding them to the _metricsToInclude_ list. Check the monitor configuration file to see if a flag is required for gathering custom metrics.

```
sh
metricsToInclude:
  - metricNames:
    - name1
    - name2
    monitorType: <name>
```

## Dimensions

The following dimensions may occur on metrics emitted by this monitor. Some dimensions may be specific to certain metrics; other dimensions can be configured. You can add extra dimensions to most metrics. The Common Configuration options page [here](https://docs.signalfx.com/en/latest/integrations/agent/monitor-config.html) also describes how to configure for these extra dimensions. 

| Name | Description |
| ---  | ---         |
| `container_id` | The ID of the running container |
| `container_image` | The container image name |
| `container_name` | The container name as it appears in the pod spec, the same as container_spec_name but retained for backwards compatibility. |
| `container_spec_name` | The container name as it appears in the pod spec |





_TROUBLESHOOTING TAB_



# Troubleshooting 

- [Confirm Installation](#confirm-installation)
- [Troubleshooting Monitor Operation](#troubleshooting-monitor-operation)

## Confirm Installation

To confirm your installation is functioning properly enter:

<This is troubleshooting the monitor installation.>

The response you see is:


## Troubleshooting Monitor Operation

<This is troubleshooting the monitor functioning.>





_USAGE TAB_




# Monitor Usage 

- [How to](#how-to) 
- [Sample code for the how to](#sample-code-for-the-how-to])

## How To

<Examples of how to use the monitor, dashboards, and metrics for a meaningful task. Discussion. >


## Sample code for the how to

<This is where you can put the sample coding that matches the "how to" section above.>













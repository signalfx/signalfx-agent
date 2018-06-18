# Quickstart

- [SingleHost](#single-host)


## Single Host

This tutorial assumes you are starting fresh and have no existing collectd agent running on your instance.

#### Step 1: Download and install the agent

```sh
curl -sSL https://dl.signalfx.com/signalfx-agent.sh > /tmp/signalfx-agent.sh
sudo sh /tmp/signalfx-agent.sh YOUR_SIGNALFX_API_TOKEN
```

Your SignalFx API Token can be obtained from the Organization->Access Token tab in [SignalFx](https://app.signalfx.com)

More detailed installation steps to install via a config management tool or using a containerized agent can be found [here](../README.md#installation)

#### Step 2: Configuration

The default configuration file should be located at `/etc/signalfx/agent.yaml`
Also, by default, the token file should be located at `/etc/signalfx/token`
which should contain your SignalFx API Token. 
In the example configuration below, the default location for the token file is being referenced.

Here is an example of the configuration file - 

```
---
# *Required* The access token for the org that you wish to send metrics to.
signalFxAccessToken: {"#from": "/etc/signalfx/token"}
ingestUrl: {"#from": "/etc/signalfx/ingest_url", default: "https://ingest.signalfx.com"}

intervalSeconds: 10

logging:
  # Valid values are 'debug', 'info', 'warning', and 'error'
  level: info

# observers are what discover running services in the environment
observers:
  - type: host

monitors:
  - {"#from": "/etc/signalfx/monitors/*.yaml", flatten: true, optional: true}
  - type: collectd/cpu
  - type: collectd/cpufreq
  - type: collectd/df
  - type: collectd/disk
  - type: collectd/interface
  - type: collectd/load
  - type: collectd/memory
  - type: collectd/protocols
  - type: collectd/signalfx-metadata
  - type: collectd/uptime
  - type: collectd/vmem
  
metricsToExclude:
```

You can add more [monitors](./monitor-config.md) and configure the monitors appropriately. 

##### Example of adding a new monitor 

To start collecting apache metrics you'd add the [apache monitor](./monitors/collectd-apache.md) to the agent.yaml file.
Your monitor list would now look similar to: 

```
monitors:
  - type: collectd/cpu
  .
  .
  .
  - type: collectd/apache
    host: localhost
    port: 80
```

##### Example of adding a new observer

To start collecting docker container metrics the first step would be to add a [docker observer](./observers/docker.md) and 
the next step would be to add a [docker metrics monitor](./monitors/docker-container-stats.md) to the agent.yaml file.

Your observer list would now look similar to:

```
observers:
  - type: host
  - type: docker
```

And your type list would now include the [docker metrics monitor](./monitors/docker-container-stats.md): 

```
monitors:
  - type: collectd/cpu
  - type: collectd/cpufreq
  .
  .
  .
  - type: docker-container-stats
```  

Any changes to the configuration file will automatically get picked up by the agent and will not require a restart.

For troubleshooting, you can also check the status of the agent: 

```
sudo signalfx-agent status
```

#### Step 3: Login to [SignalFx](https://app.signalfx.com) and see your data!



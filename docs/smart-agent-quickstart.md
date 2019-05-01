# Smart Agent Quick Start

The steps to begin using Smart Agent and links to the content are outlined below.

### Step 1: Download and install the agent on a single host.

See [Smart Agent Quick Install](#./smart-agent-quick-install.md) for Step 1, 2, & 3.

### Step 2: Confirm the installation is functioning.

#### Troubleshoot any discrepancies.

### Step 3: Login to SignalFx and discover your data.

Advanced options

### Step 4: Deploy Smart Agent on multiple hosts.

See [Smart Agent Next Steps](#./smart-agent-next-steps.md)

### Step 5: Configure various monitors to output metrics to Smart Agent. 

- To [add a new monitor](#Monitors)
- For [Windows monitor configurations](#https://docs.signalfx.com/en/latest/integrations/agent/windows.html)
- For [Linux monitor configurations](#https://docs.signalfx.com/en/latest/integrations/agent/monitor-config.html)
- For [common configuration options](#https://docs.signalfx.com/en/latest/integrations/agent/monitor-config.html) 

#### Configure optional metrics for your monitors.

See individual monitor pages for you monitor from the lists in Step 5.

### Step 6: Add a new observer to your agent configuration.

- See [Observers](#observers)

### Step 7: Explore Dashboards to display and compare data from the various sources.

See [Dashboards](#https://docs.signalfx.com/en/latest/dashboards/index.html)

## Some advanced options content

### Monitors 

You can add more [monitors](./monitor-config.md) and configure them as appropriate.

#### Example of adding a new monitor

To start collecting apache metrics, you would add the [apache monitor](./monitors/collectd-apache.md) to the agent.yaml file.
Your monitor list would then look similar to this:

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

### Observers

#### Example of adding a new observer

To start collecting docker container metrics, first add a [docker observer](./observers/docker.md).

Your observer list would then look similar to this:

```
observers:
  - type: host
  - type: docker
```

Next, add a [docker metrics monitor](./monitors/docker-container-stats.md) to the agent.yaml file. Your type list would now include this monitor (docker-container-stats) as shown below:

```
monitors:
  - type: collectd/cpu
  - type: collectd/cpufreq
  .
  .
  .
  - type: docker-container-stats
```

The agent automatically picks up any changes to the configuration file, so a restart is not required.




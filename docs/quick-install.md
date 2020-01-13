<!--- OVERVIEW --->
# Quick Install


 - [Overview](#overview)
 - [Requirements](#requirements)
 - [Installation](#installation)

## Overview


The SignalFx Smart Agent is a metric agent written in Go for monitoring infrastructure and application services in a variety of environments. It is a successor to our previous [collectd agent](https://github.com/signalfx/collectd), and still uses collectd internally on Linux; any existing Python or C-based collectd plugins will still work without modification. On Windows collectd is not included, but the agent can run python-based collectd plugins without collectd. C-based collectd plugins are not available on Windows.

 - [Concepts](#concepts)
 - [Installation](#installation)


## Concepts


## Requirements


* _Monitors_ that collect metrics from the host and applications

* _Observers_ that discover applications and services running on the host

* a _Writer_ that sends the metrics collected by monitors to SignalFx


### Monitors

Monitors collect metrics from the host system and services.  They are
configured under the `monitors` list in the agent config.  For
application specific monitors, you can define discovery rules in your monitor
configuration. A separate monitor instance is created for each discovered
instance of applications that match a discovery rule. See [Endpoint
Discovery](./auto-discovery.md) for more information.

Many of the monitors are built around [collectd](https://collectd.org), an open
source third-party monitor, and use it to collect metrics. Some other monitors
do not use collectd. However, either type is configured in the same way.

## Installation


The agent is primarily intended to monitor services/applications running on the
same host as the agent.  This is in keeping with the collectd model.  The main
issue with monitoring services on other hosts is that the `host` dimension that
collectd sets on all metrics will currently get set to the hostname of the
machine that the agent is running on.  This allows everything to have a
consistent `host` dimension so that metrics can be matched to a specific
machine during metric analysis.


#### Install on Linux
    
##### Option 1: Install from the SignalFx UI    


Observers watch the various environments that we support to discover running
services and automatically configure the agent to send metrics for those
services.

For a list of supported observers and their configurations,
see [Observer Config](./observer-config.md).

##### Option 2: Install from the documentation site 




## Installation


***

#### Install on Windows

##### Option 1: Install from the SignalFx UI    
If you are reading this content from the SignalFx Smart Agent tile in the Integrations page, then simply copy and paste the following code into your command line. (The code within the tile is already populated with your realm and your organization's access token.)


- Get your API_TOKEN from: __Organization Settings => Access Token__ tab in the SignalFx application.

- Determine YOUR\_SIGNAL_FX_REALM from your [profile page](https://docs.signalfx.com/en/latest/getting-started/get-around-ui.html#user-profile-avatar-and-color-theme) in the SignalFx web application.

To install the Smart Agent on a single Linux host, enter:

```sh
curl -sSL https://dl.signalfx.com/signalfx-agent.sh > /tmp/signalfx-agent.sh
sudo sh /tmp/signalfx-agent.sh --realm YOUR_SIGNALFX_REALM YOUR_SIGNALFX_API_TOKEN
```

__Windows:__ Ensure that the following dependencies are installed:


##### Option 2: Install from the documentation site 
If you are reading this content from the SignalFx documentation site, then SignalFx recommends that you access the Integrations page in the SignalFx UI to copy the pre-populated installation code.  


[Visual C++ Compiler for Python 2.7](https://www.microsoft.com/EN-US/DOWNLOAD/DETAILS.ASPX?ID=44266)

* Get your API\_TOKEN from:  __Organization Settings => Access Token__ tab in the SignalFx application.


The agent will be installed as a Windows service and will log to the Windows Event Log.


The agent will be installed as a Windows service and will log to the Windows Event Log.


### Step 2. Confirm your installation

To confirm the SignalFx Smart Agent installation is functional on either platform, enter:

```sh
sudo signalfx-agent status
```

The response you will see is similar to the one below:

```sh
SignalFx Agent version:           4.7.6
Agent uptime:                     8m44s
Observers active:                 host
Active Monitors:                  16
Configured Monitors:              33
Discovered Endpoint Count:        6
Bad Monitor Config:               None
Global Dimensions:                {host: my-host-1}
Datapoints sent (last minute):    1614
Events Sent (last minute):        0
Trace Spans Sent (last minute):   0
```

To verify the installation, you can run the following commands:

```sh
signalfx-agent status config - show resolved config in use by agent
signalfx-agent status endpoints - show discovered endpoints
signalfx-agent status monitors - show active monitors
signalfx-agent status all - show everything
```

#### Troubleshoot any discrepancies in the Installation

##### Realm

By default, the Smart Agent will send data to the us0 realm. If you are not in this realm, you will need to explicitly set the signalFxRealm option in your config like this:


```sh
signalFxRealm: YOUR_SIGNALFX_REALM
```


To determine if you are in a different realm and need to explicitly set the endpoints, check your profile page in the SignalFx web application.

_Configure your endpoints_

If you want to explicitly set the ingest, API server, and trace endpoint URLs, you can set them individually like so:


```sh
ingestUrl: "https://ingest.YOUR_SIGNALFX_REALM.signalfx.com"
apiUrl: "https://api.YOUR_SIGNALFX_REALM.signalfx.com"
traceEndpointUrl: "https://ingest.YOUR_SIGNALFX_REALM.signalfx.com/v1/trace"
```

This will default to the endpoints for the realm configured in signalFxRealm if not set.

To troubleshoot your installation further, check the FAQ about troubleshooting [here](./faq.md).


### Step 3. Login to SignalFx and discover your data displays.

Installation is complete.

To continue your exploration of SignalFx Smart Agent capabilities, see [Advanced Installation Options](./advanced-install-options.md).

To learn more about how your data is presented in SignalFx, see the [15-Minute SignalFx Quick Start](https://docs.signalfx.com/en/latest/getting-started/quick-start.html).

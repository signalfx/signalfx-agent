<!--- OVERVIEW --->
# Quick Install

- [Concepts](#concepts)
- [Installation](#installation)


## Smart Agent Overview

The SignalFx Smart Agent is a metric agent written in Go that is used to monitor infrastructure and application services from a variety of environments.

The Smart Agent contains three main components:

| Component | Description |
|-----------|-------------|
| Monitors  |  This component collects metrics from the host and applications. For a list of supported monitors and their configurations, see [Monitor Config](./monitor-config.md).            |
| Observers |   This component watches supported environments to discover running
services and automatically configure the Smart Agent to send metrics for those
services. For a list of supported observers and their configurations, see [Observer Config](./observer-config.md).           |
| Writer    |   The writer collects metrics from configured monitors and then sends these metrics to SignalFx on a regular basis. If you are expecting your monitors to send large volumes of metrics through a single agent, then you must update the configurations. To learn more, see [Agent Configurations](./config-schema.md#writer).          |




## Concepts

The agent has three main components:

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


For a list of supported monitors and their configurations,
see [Monitor Config](./monitor-config.md).

The agent is primarily intended to monitor services/applications running on the
same host as the agent.  This is in keeping with the collectd model.  The main
issue with monitoring services on other hosts is that the `host` dimension that
collectd sets on all metrics will currently get set to the hostname of the
machine that the agent is running on.  This allows everything to have a
consistent `host` dimension so that metrics can be matched to a specific
machine during metric analysis.

### Observers

Observers watch supported environments to discover running
services and automatically configure the Smart Agent to send metrics for those
services.

For a list of supported observers and their configurations,
see [Observer Config](./observer-config.md).

### Writer
The writer collects metrics from configured monitors and then sends these metrics to SignalFx on a regular basis. If you are expecting your monitors to send large volumes of metrics through a single agent, then you must update the configurations. To learn more, see
[Agent Configurations](./config-schema.md#writer).

## Review pre-installation requirements for the Smart Agent

Before you attempt to download and install the Smart Agent on a **single** host, review the requirements below.

(For other installation options, including bulk deployments, see [Advanced Installation Options](./advanced-install-options.md).)

| General requirements   |     Linux requirements      |  Windows requirements |
|----------|:-------------:|------:|
| <p>You must have access to your command line interface.</p> <p>You must uninstall or disable any previously installed collector agent from your host, such as collectd.</p>| <p>You must run kernel version 2.6 or higher for your Linux distribution.</p> <p>The Smart Agent is bundled with additional required dependencies, including a Java JRE runtime and a Python runtime. As a result, there is no need to proactively install additional dependencies.</p>| <p>You must run .Net Framework 3.5 on Windows 8 or higher.</p> <p>You must run Visual C++ Compiler for Python 2.7.</p>  |


## Install the Smart Agent

### Step 1. Install the SignalFx Smart Agent on Single Host

<details>
<summary>Linux</summary>
<br>

<p>For easier deployment, SignalFX recommends that you access the *SignalFX Smart Agent* tile from the *Integrations* page to copy the pre-populated installation code.</p>

**If you are reading this document directly from the Integrations page,** then simply copy and paste the following code into your command line. (The code within the tile is already populated with your *realm* and your organization's *access token*.)

```sh
curl -sSL https://dl.signalfx.com/signalfx-agent.sh > /tmp/signalfx-agent.sh
sudo sh /tmp/signalfx-agent.sh --realm YOUR_SIGNALFX_REALM YOUR_SIGNALFX_API_TOKEN
```

**If you are reading this document from the SignalFX documentation site,** then SignalFX recommends that you access the *Integrations* page to locate the installation code:  

1. Log into SignalFx, and in the top navigation bar, click *Integrations*.
2. Under *Essential Services*, click *SignalFX Smart Agent*.
3. Click *Setup*.
4. Locate the code box for *Linux* users.
5. Copy and paste the code into your command line to run. (The code within the tile is already populated with your *realm* and your organization's *access token*.)  
</details>


<details>
<summary>Windows</summary>
<br>

<p>For easier deployment, SignalFX recommends that you access the *SignalFX Smart Agent* tile from the *Integrations* page to copy the pre-populated installation code.</p>

**If you are reading this document directly from the Integrations page,** then simply copy and paste the following code into your command line. (The code within the tile is already populated with your *realm* and your organization's *access token*.)</p>  

```sh
& {Set-ExecutionPolicy Bypass -Scope Process -Force; $script = ((New-Object System.Net.WebClient).DownloadString('https://dl.signalfx.com/signalfx-agent.ps1')); $params = @{access_token = "YOUR_SIGNALFX_API_TOKEN"};;
ingest_url = "https://ingest.YOUR_SIGNALFX_REALM.signalfx.com"; api_url = "https://api.YOUR_SIGNALFX_REALM.signalfx.com"}; Invoke-Command -ScriptBlock ([scriptblock]::Create(". {$script} $(&{$args} @params)"))}
```

If you are reading this document from the SignalFX documentation site, then SignalFX recommends that you access the *Integrations* page to locate the installation code:  

1. Log into SignalFx, and in the top navigation bar, click *Integrations*.
2. Under *Essential Services*, click *SignalFX Smart Agent*.
3. Click *Setup*.
4. Locate the code box for *Linux* users.
5. Copy and paste the code into your command line to run. (The code within the tile is already populated with your *realm* and your organization's *access token*.)  

The agent will be installed as a Windows service and will log to the Windows Event Log.
</details>

***

### Step 2. Confirm your installation

1. Enter the following command to confirm that your Smart Agent is functional:

```sh
sudo signalfx-agent status
```

The return should be similar to the following example:  

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

2. Enter the following commands to verify the installation:

| Command | Description   |
|---|---|
| <code>signalfx-agent status config</code>   | This command shows resolved config in use by the Smart Agent. |
| <code>signalfx-agent status endpoints</code>  | This command shows discovered endpoints.  |
| <code>signalfx-agent status monitors</code>  | This command shows active monitors.  |
| <code>signalfx-agent status all</code>  | This command shows the above statuses. |

***

### Troubleshoot the Smart Agent installation

If you are unable to install the Smart Agent, consider the following issues:

* You may need to update your realm. By default, the Smart Agent will send data to the us0 realm. If you are not in this realm, you will need to set the signalFxRealm option with your actual realm :


```sh
signalFxRealm: YOUR_SIGNALFX_REALM
```

To find your realm:
1. In SignalFx, in the top, right corner, click your profile icon.
2. Click **My Profile**.
3. Next to **Organizations**, review the listed realm.

***

* You may need to set the endpoints. To explicitly set the ingest, API server, and trace endpoint URLs, review the following examples:  

```sh
ingestUrl: "https://ingest.YOUR_SIGNALFX_REALM.signalfx.com"
apiUrl: "https://api.YOUR_SIGNALFX_REALM.signalfx.com"
traceEndpointUrl: "https://ingest.YOUR_SIGNALFX_REALM.signalfx.com/v1/trace"
```

***

For additional installation troubleshooting information, see [Frequently Asked Questions](./faq.md).

***

### Review additional documentation

After a successful installation:

* To learn more about the capabilities of the SignalFx Smart Agent, see [Advanced Installation Options](./advanced-install-options.md).

* To learn more about the SignalFX, including how data is displayed, see the [15-Minute SignalFx Quick Start](https://docs.signalfx.com/en/latest/getting-started/quick-start.html).

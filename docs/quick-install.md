<!--- OVERVIEW --->
# Quick Install

## Smart Agent Overview

The SignalFx Smart Agent is a metric-based agent written in Go that is used to monitor infrastructure and application services from a variety of environments.

The Smart Agent contains three main components:

| Component | Description |
|-----------|-------------|
| Monitors  |  This component collects metrics from the host and applications. For a list of supported monitors and their configurations, see [Monitor Config](./monitor-config.md).            |
| Observers |   This component collects metrics from services that are running in an environment. For a list of supported observers and their configurations, see [Observer Config](./observer-config.md).           |
| Writer    |   This component collects metrics from configured monitors and then sends these metrics to SignalFx on a regular basis. If you are expecting your monitors to send large volumes of metrics through a single agent, then you must update the configurations. To learn more, see [Agent Configurations](./config-schema.md#writer).          |


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

For easier deployment, SignalFX recommends that you access the *SignalFX Smart Agent* tile from the *Integrations* page to copy the pre-populated installation code.

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

For easier deployment, SignalFX recommends that you access the *SignalFX Smart Agent* tile from the *Integrations* page to copy the pre-populated installation code.

**If you are reading this document directly from the Integrations page,** then simply copy and paste the following code into your command line. (The code within the tile is already populated with your *realm* and your organization's *access token*.)</p>  

```sh
& {Set-ExecutionPolicy Bypass -Scope Process -Force; $script = ((New-Object System.Net.WebClient).DownloadString('https://dl.signalfx.com/signalfx-agent.ps1')); $params = @{access_token = "<TOKEN>"; ingest_url = "https://ingest.<REALM>.signalfx.com"; api_url = "https://api.<REALM>.signalfx.com"}; Invoke-Command -ScriptBlock ([scriptblock]::Create(". {$script} $(&{$args} @params)"))}
```

**If you are reading this document from the SignalFX documentation site,** then SignalFX recommends that you access the *Integrations* page to locate the installation code:  

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
| <code>signalfx-agent status all</code>  | This command shows all of the above statuses. |

***

### Troubleshoot the Smart Agent installation

If you are unable to install the Smart Agent, consider the following issues:

* You may need to update your realm. By default, the Smart Agent will send data to the us0 realm. If you are not in this realm, you will need to set the signalFxRealm option with your correct realm:


```sh
signalFxRealm: YOUR_SIGNALFX_REALM
```

```sh
To find your realm:
1. In SignalFx, in the top, right corner, click your profile icon.
2. Click **My Profile**.
3. Next to **Organizations**, review the listed realm.
```

***

* You may need to set the endpoints. To explicitly set the ingest, API server, and trace endpoint URLs, review the following examples:  

```sh
ingestUrl: "https://ingest.YOUR_SIGNALFX_REALM.signalfx.com"
apiUrl: "https://api.YOUR_SIGNALFX_REALM.signalfx.com"
traceEndpointUrl: "https://ingest.YOUR_SIGNALFX_REALM.signalfx.com/v1/trace"
```

***

* Review error logs.

For Linux, you can use the following command to view logs via journalctl:

```sh
journalctl `which signalfx-agent` | tail -100
```

For Windows, simply review event logs.

For additional installation troubleshooting information, inluding how to review logs, see [Frequently Asked Questions](./faq.md).

***

### Review additional documentation

After a successful installation, you can learn more about:

* The capabilities of the SignalFx Smart Agent. See [Advanced Installation Options](./advanced-install-options.md).

* The SignalFX UI, including how data is displayed. See [15-Minute SignalFx Quick Start](https://docs.signalfx.com/en/latest/getting-started/quick-start.html).

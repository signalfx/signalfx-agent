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
  
```sh curl -sSL https://dl.signalfx.com/signalfx-agent.sh > /tmp/signalfx-agent.sh
sudo sh /tmp/signalfx-agent.sh --realm YOUR_SIGNALFX_REALM YOUR_SIGNALFX_API_TOKEN

<!-- In the above command, YOUR_SIGNALFX_REALM represents your SignalFX instance's running environment (realm). To locate your realm, in the SignalFX UI, in the top, right corner, click your profile icon, click My Profile, and then search for your realm.

Additionally, YOUR_SIGNALFX_API_TOKEN, represents your organization's default access token. To locate your organization's token, in the SignalFX UI, in the top, right corner, click your profile icon, hover over Organization Settings, click Access Tokens, search for Default, expand the field, and then click Show Access Token. -->
```  
</details>

<details>
<summary>Windows</summary>
<br>


```sh
& {Set-ExecutionPolicy Bypass -Scope Process -Force; $script = ((New-Object System.Net.WebClient).DownloadString('https://dl.signalfx.com/signalfx-agent.ps1')); $params = @{access_token = "YOUR_SIGNALFX_API_TOKEN"; ingest_url = "https://ingest.YOUR_SIGNALFX_REALM.signalfx.com"; api_url = "https://api.YOUR_SIGNALFX_REALM.signalfx.com"}; Invoke-Command -ScriptBlock ([scriptblock]::Create(". {$script} $(&{$args} @params)"))}
```

<!-- In the above command, YOUR_SIGNALFX_REALM represents your SignalFX instance's running environment (realm). To locate your realm, in the SignalFX UI, in the top, right corner, click your profile icon, click My Profile, and then search for your realm.

Additionally, YOUR_SIGNALFX_API_TOKEN, represents your organization's default access token. To locate your organization's token, in the SignalFX UI, in the top, right corner, click your profile icon, hover over Organization Settings, click Access Tokens, search for Default, expand the field, and then click Show Access Token. -->


The agent will be installed as a Windows service and will log to the Windows Event Log.
</details>

***

### Step 2. Confirm your installation

1. From the command line, enter the following command to confirm that your Smart Agent is functional:

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

***
 
#### Update your realm

By default, the Smart Agent will send data to the *us0* realm. To find your realm:

1. In SignalFx, in the top, right corner, click your profile icon.
2. Click **My Profile**.
3. Next to **Organizations**, review the listed realm.

***

#### Set the endpoints

To explicitly set the ingest, API server, and trace endpoint URLs, review the following examples:  

```sh
ingestUrl: "https://ingest.YOUR_SIGNALFX_REALM.signalfx.com"
apiUrl: "https://api.YOUR_SIGNALFX_REALM.signalfx.com"
traceEndpointUrl: "https://ingest.YOUR_SIGNALFX_REALM.signalfx.com/v1/trace"
```

***

#### Review your error logs

For Linux, use the following command to view error logs via Journal:

```sh
journalctl -u signalfx-agent | tail -100
```

For Windows, simply review the event logs.

***

For additional installation troubleshooting information, including how to review logs, see [Frequently Asked Questions](./faq.md).

***

### Review additional documentation

After a successful installation, you can learn more about:

* The capabilities of the SignalFx Smart Agent. See [Advanced Installation Options](./advanced-install-options.md).

* The SignalFx UI, including how data is displayed. See [15-Minute SignalFx Quick Start](https://docs.signalfx.com/en/latest/getting-started/quick-start.html).

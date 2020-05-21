<!--- OVERVIEW --->
# Quick Install

SignalFx Smart Agent Integration installs the Smart Agent application on a single host machine from which you want to collect monitoring data. Smart Agent collects infrastructure monitoring, µAPM, and Kubernetes data.

For other installation options, including bulk deployments, see [Advanced Installation Options](./advanced-install-options.md).

## Prerequisites

### General
- Ensure that you've installed the applications and services you want to monitor on a Linux or Windows host. SignalFx doesn't support Smart Agent on MacOS or any other OS besides Linux and Windows.
- Uninstall or disable any previously-installed collector agents from your host, such as `collectd`.
- If you have any questions about compatibility between Smart Agent and your host machine or its applications and services, contact your Splunk support representative.

### Linux
- Ensure that you have access to `terminal` or a similar command line interface application.
- Ensure that your Linux username has permission to run the following commands: 
    - `curl` 
    - `sudo`
- Ensure that your machine is running Linux kernel version 2.6 or higher.

### Windows
- Ensure that you have access to Windows PowerShell 6
- Ensure that your machine is running Windows 8 or higher.
- Ensure that .Net Framework 3.5 or higher is installed.

## Steps

### Access the SignalFx UI

This content appears in both the documentation site and in the SignalFx UI.

If you are reading this content from the documentation site, please access the SignalFx UI so that you can paste pre-populated commands. 

To access this content from the SignalFx UI:
1. In the SignalFx UI, in the top menu, click **Integrations**. 
2. Locate and select **SignalFx SmartAgent**. 
3. Click **Setup**, and continue reading the instructions. 

### Install Signalfx Smart Agent on Linux

This section lists the steps for installing SignalFx Smart Agent on Linux. If you want to install it on Windows, proceed to the next section, **Install SignalFx Smart Agent on Windows**.

1. Open your command line application.

2. Download the Smart Agent install script from its repository location. Copy the following command and paste it into the window of `terminal` or a similar app.
`curl -sSL https://dl.signalfx.com/signalfx-agent.sh > /tmp/signalfx-agent.sh;`

If the download succeeds, you don't see any output, but you do see a new command prompt.

3. When the download finishes, run the install script in a command line window. Copy the following command and paste it into the window of `terminal` or a similar app.
`sudo sh /tmp/signalfx-agent.sh --realm YOUR_SIGNALFX_REALM -- YOUR_SIGNALFX_API_TOKEN`

When this command finishes, it displays the following:

`The SignalFx Agent has been successfully installed.`

`Make sure that your system's time is relatively accurate or else datapoints may not be accepted.`

`The agent's main configuration file is located at /etc/signalfx/agent.yaml.`

4. (Optional) If you want to override the default user and group names, review your [deployment's README file](https://github.com/signalfx/signalfx-agent/tree/master/deployments) and locate the option to set the user/group ownership for the signalfx-agent service.

5. If your installation succeeds, proceed to the section **Verify Your Installation**. Otherwise, see the section **Troubleshoot Your Installation**.

### Install SignalFx Smart Agent on Windows

1. Run Windows PowerShell.

2. To configure Windows execution policy, download the Smart Agent install script, and run it, copy the following commands and paste them into the Windows PowerShell window:

```sh
& {Set-ExecutionPolicy Bypass -Scope Process -Force; $script = ((New-Object System.Net.WebClient).DownloadString('https://dl.signalfx.com/signalfx-agent.ps1')); $params = @{access_token = "YOUR_SIGNALFX_API_TOKEN"; ingest_url = "https://ingest.YOUR_SIGNALFX_REALM.signalfx.com"; api_url = "https://api.YOUR_SIGNALFX_REALM.signalfx.com"}; Invoke-Command -ScriptBlock ([scriptblock]::Create(". {$script} $(&{$args} @params)"))}
```

The install script starts the agent as a Windows service that writes messages to the Windows Event Log.

## Verify Your Installation

1. To verify that you've successfully installed the SignalFx Smart Agent, run the following command in a command line interface application.

**HINT:** Copy the command and paste it into the application window.

**For Linux:** Use `terminal` or a similar app 

**For Windows:** Use Windows PowerShell

`sudo signalfx-agent status`

The command displays output that is similar to the following:

    ```sh
    SignalFx Agent version:           5.1.0
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

2. To perform additional verification, run each of the following commands in your command line interface application:

- `signalfx-agent status config:`
Displays the current Smart Agent configuration

- `signalfx-agent status endpoints`
Shows endpoints discovered by Smart Agent

- `signalfx-agent status monitors`
Shows Smart Agent's active monitors. These plugins poll apps and services to retrieve data.

## Troubleshoot Smart Agent Installation
If the Smart Agent installation fails, use the following procedures to gather troubleshooting information.

### General troubleshooting
To learn how to review signalfx-agent logs, see [Frequently Asked Questions](./faq.md).

### Linux troubleshooting
To view the most recent 100 error logs that signalfx-agent has written to the systemd journal, run the following command in terminal or a similar application:

`journalctl -u signalfx-agent | tail -100`

### Windows troubleshooting
Run **Administrative Tools > Event Viewer** to view signalfx-agent error logs.


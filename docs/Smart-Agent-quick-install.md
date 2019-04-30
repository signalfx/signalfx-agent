# SignalFx Smart Agent Quick Install

[![GoDoc](https://godoc.org/github.com/signalfx/signalfx-agent?status.svg)](https://godoc.org/github.com/signalfx/signalfx-agent)
[![CircleCI](https://circleci.com/gh/signalfx/signalfx-agent.svg?style=shield)](https://circleci.com/gh/signalfx/signalfx-agent)


 - [Installation](#installation)
 - [Confirmation](#confirmation)
 - [Troubleshooting](#troubleshooting-the-installation)
 - [Next Steps](#next-steps)
 - [Other methods of Installation](#other-methods-of-installation)
 

## Installation

### Single Host

The installation statement includes YOUR_\SIGNALFX\_REALM. If this value is not set, SignalFx assumes your organization is in the us0 realm. To determine if you are in a different realm and must supply your realm value in the statement, check your profile page in the SignalFx web application.

Your SignalFx API Token can be obtained from the Organization->Access Token tab in SignalFx.

#### Linux

The Smart Agent for Linux contains a Java JRE runtime and a Python runtime, so there are no
additional dependency requirements. 

Uninstall any existing collectd agent before installing SignalFx Smart Agent.  

To install the Smart Agent on a single Linux host, enter:

```sh
curl -sSL https://dl.signalfx.com/signalfx-agent.sh > /tmp/signalfx-agent.sh
sudo sh /tmp/signalfx-agent.sh --realm YOUR_SIGNALFX_REALM YOUR_SIGNALFX_API_TOKEN
```

#### Windows

The Smart Agent for Windows has these two dependencies:

- [.Net Framework 3.5](https://docs.microsoft.com/en-us/dotnet/framework/install/dotnet-35-windows-10) (Windows 8+)
- [Visual C++ Compiler for Python 2.7](https://www.microsoft.com/EN-US/DOWNLOAD/DETAILS.ASPX?ID=44266)

To install the Smart Agent on a single Windows host, enter:

```sh
& {Set-ExecutionPolicy Bypass -Scope Process -Force; $script = ((New-Object System.Net.WebClient).DownloadString('https://dl.signalfx.com/signalfx-agent.ps1')); $params = @{access_token = "YOUR_SIGNALFX_API_TOKEN"}; Invoke-Command -ScriptBlock ([scriptblock]::Create(". {$script} $(&{$args} @params)"))}`
```

The agent will be installed as a Windows service and will log to the Windows Event Log.

## Confirmation

To confirm the SignalFx Smart Agent installation is functional, for Linux enter:

```sh
sudo signalfx-agent status
```

The response you will see is --

To confirm the SignalFx Smart Agent installation is functional, for Windows enter:

```sh
something
```

The response you will see is ---

__Now login to SignalFx to see your data!__

## Troubleshooting the Installation

To troubleshoot the Linux installation -- 

To troubleshoot the Windows installation -- 

### Realm

By default, the Smart Agent will send data to the us0 realm. If you are not in this realm, you will need to explicitly set the signalFxRealm option in your config like this:
```sh
signalFxRealm: YOUR_SIGNALFX_REALM
```
To determine if you are in a different realm and need to explicitly set the endpoints, check your profile page in the SignalFx web application. Click the Avatar in the upper right of the application to access your profile.

## Next Steps

To familiarize yourself with Smart Agent, you can explore the various topics in [Smart Agent Quick Start](/docs/smart-agent-quick-start.md).

To install Smart Agent on multiple hosts using configuration management tools or packages, go to the Smart Agent Next Steps information [here](/docs/smart-agent-next-steps.md).

For Windows, you may want to configure Monitors included in Smart Agent. More information is [here](https://docs.signalfx.com/en/latest/integrations/agent/windows.html).

For Linux, to configure monitors for use with Smart Agent in your environment, go to [Monitor Configuration](https://docs.signalfx.com/en/latest/integrations/agent/monitor-config.html).

## Other methods of Installation

### Linux Standalone tar.gz

See the Smart Agent [README](https://github.com/signalfx/signalfx-agent/blob/master/README.md) file for information.

### Windows Standalone .zip

See the Smart Agent [README](https://github.com/signalfx/signalfx-agent/blob/master/README.md) file for information.






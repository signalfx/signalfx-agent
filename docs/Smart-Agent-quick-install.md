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

Your SignalFx API Token can be obtained from the Organization->Access Token tab in SignalFx.

Certain installation statements include YOUR_SIGNALFX_REALM. A realm is a self-contained deployment of SignalFx in which your organization is hosted. Different realms have different API endpoints (e.g. the endpoint for sending data is ingest.us1.signalfx.com for the us1 realm, and ingest.eu0.signalfx.com for the eu0 realm). If this value is not set, SignalFx assumes your organization is in the us0 realm. To determine if you are in a different realm and need to supply your realm value in those statements, check your profile page in the SignalFx web application.

The Smart Agent for Linux contains a Java JRE runtime and a Python runtime, so there are no
additional dependency requirements. 

Uninstall any existing collectd agent before installing SignalFx Smart Agent.  

To install the Smart Agent on a single Linux host, enter:

```sh
curl -sSL https://dl.signalfx.com/signalfx-agent.sh > /tmp/signalfx-agent.sh
sudo sh /tmp/signalfx-agent.sh --realm YOUR_SIGNALFX_REALM YOUR_SIGNALFX_API_TOKEN
```

The Smart Agent for Windows has these two dependencies:

- [.Net Framework 3.5](https://docs.microsoft.com/en-us/dotnet/framework/install/dotnet-35-windows-10) (Windows 8+)
- [Visual C++ Compiler for Python 2.7](https://www.microsoft.com/EN-US/DOWNLOAD/DETAILS.ASPX?ID=44266)

To install the Smart Agent on a single Windows host, enter:

```sh
& {Set-ExecutionPolicy Bypass -Scope Process -Force; $script = ((New-Object System.Net.WebClient).DownloadString('https://dl.signalfx.com/signalfx-agent.ps1')); $params = @{access_token = "YOUR_SIGNALFX_API_TOKEN"};; ingest_url = "https://ingest.YOUR_SIGNALFX_REALM.signalfx.com"; api_url = "https://api.YOUR_SIGNALFX_REALM.signalfx.com"}; Invoke-Command -ScriptBlock ([scriptblock]::Create(". {$script} $(&{$args} @params)"))}`
```

The agent will be installed as a Windows service and will log to the Windows Event Log.

## Confirmation

To confirm the SignalFx Smart Agent installation is functional on either platform, enter:

```sh
sudo signalfx-agent status
```

The response you will see is:

```sh
SignalFx Agent version:           4.5.0
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

## Troubleshooting the Installation

To troubleshoot your installation, check the FAQ about troubleshooting [here](./docs/faq.md#how-can-I-see-the-datapoints-emitted-by-the-agent-to-troubleshoot-issues).


### Realm

By default, the Smart Agent will send data to the us0 realm. If you are not in this realm, you will need to explicitly set the signalFxRealm option in your config like this:

```sh
signalFxRealm: YOUR_SIGNALFX_REALM
```
To determine if you are in a different realm and need to explicitly set the endpoints, check your profile page in the SignalFx web application.

## Next Steps

Additional information on configuration management tools and package installations is [here](/docs/smart-agent-next-steps.md).

## Other methods of Installation

See the README file [here](https://github.com/signalfx/signalfx-agent/blob/master/README.md).

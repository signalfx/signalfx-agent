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

The Smart Agent for Linux contains a Java JRE runtime and a Python runtime, so there are no
additional dependency requirements. 

Uninstall any existing collectd agent before installing SignalFx Smart Agent.  

To install the Smart Agent on a single Linux host, enter:

```sh
curl -sSL https://dl.signalfx.com/signalfx-agent.sh > /tmp/signalfx-agent.sh
sudo sh /tmp/signalfx-agent.sh YOUR_SIGNALFX_API_TOKEN
```

The Smart Agent for Windows has these two dependencies:

- [.Net Framework 3.5](https://docs.microsoft.com/en-us/dotnet/framework/install/dotnet-35-windows-10) (Windows 8+)
- [Visual C++ Compiler for Python 2.7](https://www.microsoft.com/EN-US/DOWNLOAD/DETAILS.ASPX?ID=44266)

To install the Smart Agent on a single Windows host, enter:

`& {Set-ExecutionPolicy Bypass -Scope Process -Force; $script = ((New-Object System.Net.WebClient).DownloadString('https://dl.signalfx.com/signalfx-agent.ps1')); $params = @{access_token = "YOUR_SIGNALFX_API_TOKEN"}; Invoke-Command -ScriptBlock ([scriptblock]::Create(". {$script} $(&{$args} @params)"))}`

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


## Troubleshooting the Installation

To troubleshoot the Linux installation -- 

To troubleshoot the Windows installation -- Is this where the link to "Agent Configuration: Configuring your realm" info should be?

## Next Steps

To install Smart Agent on multiple hosts using Configuration Management Tools or Packages, go to the Integrations page, and then click the icon of the tool you want to use. Additional information on Configuration Management tools and Package installations is here.

To configure monitors to use with Smart Agent in your environment, go to [Monitor Configuration](https://docs.signalfx.com/en/latest/integrations/agent/monitor-config.html).

## Other methods of Installation

### Linux Standalone tar.gz

If you don't want to use a distro package, we offer a
.tar.gz that can be deployed to the target host.  This bundle is available for
download on the [Github Releases
Page](https://github.com/signalfx/signalfx-agent/releases) for each new
release.

To use the bundle:

1) Unarchive it to a directory of your choice on the target system.

2) Ensure a valid configuration file is available somewhere on the target
system.  The main thing that the distro packages provide -- but that you will
have to provide manually with the bundle -- is a run directory for the agent to
use.  Since you aren't installing from a package, there are three config 
options that you will especially want to consider:

 - `internalStatusHost` - This is the host name that
	 the agent will listen on so that the `signalfx-agent status` command can
	 read diagnostic information from a running agent.  This is also the host name the
	 agent will listen on to serve internal metrics about the agent.  These metrics can
	 can be scraped by the `internal-metrics` monitor.  This will default to `localhost`
	 if left blank.

 - `internalStatusPort` - This is the port that the agent will listen on so that
	 the `signalfx-agent status` command can read diagnostic information from
	 a running agent.  This is also the host name the agent will listen on to serve
	 internal metrics about the agent.  These metrics can can be scraped by the
	 `internal-metrics` monitor.  This will default to `8095`.

 - `collectd.configDir` - This is where the agent writes the managed collectd
	 config, since collectd can only be configured by files.  Note that **this
	 entire dir will be wiped by the agent upon startup** so that it doesn't
	 pick up stale collectd config, so be sure that it is not used for anything
	 else.  Also note that **these files could have sensitive information in
	 them** if you have passwords configured for collectd monitors, so you
	 might want to place this dir on a `tmpfs` mount to avoid credentials 
	 persisting on disk.

When using the [host observer](./docs/observers/host.md), the agent requires
the [Linux
capabilities](http://man7.org/linux/man-pages/man7/capabilities.7.html)
`DAC_READ_SEARCH` and `SYS_PTRACE`, both of which are necessary to allow the
agent to determine which processes are listening on network ports on the host.
Otherwise, there is nothing built into the agent that requires privileges.
When using a package to install the agent, the agent binary is given those
capabilities in the package post-install script, but the agent is run as the
`signalfx-agent` user.  If you are not using the `host` observer, then you can
strip those capabilities from the agent binary if desired.

You should generally not run the agent as `root` unless you can't use
capabilities for some reason.

3) Run the agent by invoking the archive path
`signalfx-agent/bin/signalfx-agent -config <path to config.yaml>`.  By default,
the agent logs only to stdout/err. If you want to persist logs, you must direct
the output to a log file or other log management system.  See the
[signalfx-agent command](./docs/signalfx-agent.1.man) doc for more information on
supported command flags.

### Windows Standalone .zip
If you don't want to use the installer script, we offer a
.zip that can be deployed to the target host.  This bundle is available for
download on the [Github Releases
Page](https://github.com/signalfx/signalfx-agent/releases) for each new
release.

Before proceeding make sure the following requirements are met.
- [.Net Framework 3.5](https://docs.microsoft.com/en-us/dotnet/framework/install/dotnet-35-windows-10) (Windows 8+)
- [Visual C++ Compiler for Python 2.7](https://www.microsoft.com/EN-US/DOWNLOAD/DETAILS.ASPX?ID=44266)

On Windows the agent must be installed and run under an administrator account. To use the bundle:

1) Unzip it to a directory of your choice on the target system.

2) Ensure a valid configuration file is available somewhere on the target
system.  The main thing that the installer script provides -- but that you will
have to provide manually with the bundle -- is a run directory for the agent to
use.  Since you aren't installing from a package, there are two config
options that you will especially want to consider:

 - `internalStatusHost` - This is the host name that
	 the agent will listen on so that the `signalfx-agent status` command can
	 read diagnostic information from a running agent.  This is also the host name the
	 agent will listen on to serve internal metrics about the agent.  These metrics can
	 can be scraped by the `internal-metrics` monitor.  This will default to `localhost`
	 if left blank.

 - `internalStatusPort` - This is the port that the agent will listen on so that
	 the `signalfx-agent status` command can read diagnostic information from
	 a running agent.  This is also the host name the agent will listen on to serve
	 internal metrics about the agent.  These metrics can can be scraped by the
	 `internal-metrics` monitor.  This will default to `8095`.

3) Run the agent by invoking the agent executable
`SignalFxAgent\bin\signalfx-agent.exe -config <path to config.yaml>`.  By default,
the agent logs only to stdout/err. If you want to persist logs, you must direct
the output to a log file or other log management system.  See the
[signalfx-agent command](./docs/signalfx-agent.1.man) doc for more information on
supported command flags.

4) You may optionally install the agent as a Windows service by invoking the
agent executable and specifying a few command line flags.  The examples below
show how to do install and start the agent as a Windows service.

- Install Service

		PS> SignalFx\SignalFxAgent\bin\signalfx-agent.exe -service "install" -logEvents -config <path to config file>

- Start Service

		PS> SignalFx\SignalFxAgent\bin\signalfx-agent.exe -service "start"









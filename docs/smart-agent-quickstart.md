# Smart Agent Quick Start

- [Deploy Directly on Host](#deploy-directly-on-host)


## Deploy Directly on Host

This tutorial assumes you are starting fresh and have no existing collectd agent running on your instance.

#### Step 1: Download and install the agent

##### Linux

```sh
curl -sSL https://dl.signalfx.com/signalfx-agent.sh > /tmp/signalfx-agent.sh
sudo sh /tmp/signalfx-agent.sh YOUR_SIGNALFX_API_TOKEN
```

##### Windows

Ensure that the folowing dependencies are installed:
- [.Net Framework 3.5](https://docs.microsoft.com/en-us/dotnet/framework/install/dotnet-35-windows-10) (Windows 8+)
- [Visual C++ Compiler for Python 2.7](https://www.microsoft.com/EN-US/DOWNLOAD/DETAILS.ASPX?ID=44266)

Once the dependencies have been installed, use the following powershell script
to install the agent.  The agent will be installed as a Windows service and will
log to the Windows Event Log.

```sh
& {Set-ExecutionPolicy Bypass -Scope Process -Force; $script = ((New-Object System.Net.WebClient).DownloadString('https://dl.signalfx.com/signalfx-agent.ps1')); $params = @{access_token = "YOUR_SIGNALFX_API_TOKEN"}; Invoke-Command -ScriptBlock ([scriptblock]::Create(". {$script} $(&{$args} @params)"))}
```

Your SignalFx API Token can be obtained from the Organization->Access Token tab in [SignalFx](https://app.signalfx.com).

More detailed installation steps to install via a config management tool or using a containerized agent can be found [here](../README.md#installation).

#### Step 2: Configuration

The default configuration file should be located at `/etc/signalfx/agent.yaml` on Linux
and `\ProgramData\SignalFxAgent\agent.yaml` on Windows.
Also, by default, the file containing your SignalFx API token should be located at
`/etc/signalfx/token` on Linux and `\ProgramData\SignalFxAgent\token` on Windows.

In the referenced example agent.yaml configuration files below, the default
location for the token file is used.

- [Linux Default Configuration File](https://github.com/signalfx/signalfx-agent/blob/master/packaging/etc/agent.yaml)

- [Windows Default Configuration File](https://github.com/signalfx/signalfx-agent/blob/master/packaging/win/agent.yaml)

##### Configure your endpoints

By default, the Smart Agent will send data to the `us0` realm.
If you are not in this realm, you will need to explicitly set the
`signalFxRealm` option in your config like this:

```
signalFxRealm: <MY REALM>
```

To determine if you are in a different realm and need to
explicitly set the endpoints, check your profile page in the SignalFx
web application.

If you want to explicitly set the ingest, API server, and trace endpoint URLs,
you can set them individually like so:

```
ingestUrl: "https://ingest.{REALM}.signalfx.com"
apiUrl: "https://api.{REALM}.signalfx.com"
traceEndpointUrl: "https://ingest.{REALM}.signalfx.com/v1/trace"
```

They will default to the endpoints for the realm configured in `signalFxRealm`
if not set.

You can add more [monitors](./monitor-config.md) and configure them as appropriate.

##### Example of adding a new monitor

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

##### Example of adding a new observer

To start collecting docker container metrics, your first step would be to add a [docker observer](./observers/docker.md).

Your observer list would then look similar to this:

```
observers:
  - type: host
  - type: docker
```

Next, you would add a [docker metrics monitor](./monitors/docker-container-stats.md) to the agent.yaml file. Your type list would now include this monitor (docker-container-stats):

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

For troubleshooting, you can also check the status of the agent:

```
sudo signalfx-agent status
```

#### Step 3: Log into [SignalFx](https://app.signalfx.com) and see your data!



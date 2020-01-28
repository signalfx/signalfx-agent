# SignalFx Smart Agent 

[![GoDoc](https://godoc.org/github.com/signalfx/signalfx-agent?status.svg)](https://godoc.org/github.com/signalfx/signalfx-agent)
[![CircleCI](https://circleci.com/gh/signalfx/signalfx-agent.svg?style=shield)](https://circleci.com/gh/signalfx/signalfx-agent)

The SignalFx Smart Agent is a metric agent written in Go for monitoring
infrastructure and application services in a variety of different environments.
It is meant as a successor to our previous [collectd
agent](https://github.com/signalfx/collectd), but still uses that internally on Linux --
so any existing Python or C-based collectd plugins will still work without
modification.  On Windows collectd is not included, but the agent is capable of
running python based collectd plugins without collectd.  C-based collectd plugins
are not available on Windows.

 - [Concepts](#concepts)
 - [Installation](#installation)
 - [Configuration](#configuration)
 - [Logging](#logging)
 - [Proxy Support](#proxy-support)
 - [Diagnostics](#diagnostics)
 - [Development](#development)

## Concepts

The agent has three main components:

1) _Observers_ that discover applications and services running on the host
2) _Monitors_ that collect metrics, events, and dimension properties the host and applications
3) The _Writer_ that sends the metrics, events, and dimension updates collected by monitors to SignalFx.

### Observers

Observers watch the various environments that we support to discover running
services and automatically configure the agent to send metrics for those
services.

For a list of supported observers and their configurations,
see [Observer Config](./docs/observer-config.md).

### Monitors

Monitors collect metrics from the host system and services.  They are
configured under the `monitors` list in the agent config.  For
application-specific monitors, you can define discovery rules in your monitor
configuration. A separate monitor instance is created for each discovered
instance of applications that match a discovery rule. See [Auto
Discovery](./docs/auto-discovery.md) for more information.

Many of the monitors are built around [collectd](https://collectd.org), an open
source third-party monitor, and use it to collect metrics. Some other monitors
do not use collectd. However, either type is configured in the same way.

For a list of supported monitors and their configurations,
see [Monitor Config](./docs/monitor-config.md).

The agent is primarily intended to monitor services/applications running on the
same host as the agent.  This is in keeping with the collectd model.  The main
issue with monitoring services on other hosts is that the `host` dimension that
collectd sets on all metrics will currently get set to the hostname of the
machine that the agent is running on.  This allows everything to have a
consistent `host` dimension so that metrics can be matched to a specific
machine during metric analysis.

### Writer
The writer collects metrics emitted by the configured monitors and sends them
to SignalFx on a regular basis.  There are a few things that can be
[configured](./docs/config-schema.md#writer) in the writer, but this is generally
only necessary if you have a very large number of metrics flowing through a
single agent.

## Installation

The agent is available for Linux in both a containerized and standalone form.
Whatever form you use, the dependencies are completely bundled along with the
agent, including a Java JRE runtime and a Python runtime, so there are no
additional dependencies required.  This means that the agent should work on any
relatively modern Linux distribution (kernel version 2.6+).

The agent is also available on Windows in standalone form.  It
contains its own Python runtime, but has an external depencency on the
[Visual C++ Compiler for Python 2.7](https://www.microsoft.com/EN-US/DOWNLOAD/DETAILS.ASPX?ID=44266)
in order to operate.  The agent supports Windows Server 2008 and above.

To get started deploying the Smart Agent directly on a host, see the
[Smart Agent Quick Install](./docs/quick-install.md) guide.

### Deployment
We support the following deployment/configuration management tools to automate the
installation process.  See [Bundles](#bundles) for a list of underlying
packages for the agent.

#### Installer Script
For non-containerized environments, there is a convenience script that you can
run on your host to install the agent package.  This is useful for testing and
trials, but for full-scale deployments you will probably want to use a
configuration management system like Chef or Puppet.

##### Linux
You can [view the source for the installer script](./deployments/installer/install.sh)
and use it on your hosts by running:

```sh
curl -sSL https://dl.signalfx.com/signalfx-agent.sh > /tmp/signalfx-agent.sh
sudo sh /tmp/signalfx-agent.sh --realm YOUR_SIGNALFX_REALM -- YOUR_SIGNALFX_API_TOKEN
```

##### Windows
The Agent has two dependencies on Windows which must be satisfied before running the installer script.

- [.Net Framework 3.5](https://docs.microsoft.com/en-us/dotnet/framework/install/dotnet-35-windows-10) (Windows 8+)
- [Visual C++ Compiler for Python 2.7](https://www.microsoft.com/EN-US/DOWNLOAD/DETAILS.ASPX?ID=44266)

The installer script is written for Powershell v3.0 and above and will not function correctly on earlier versions.

Once the dependencies have been installed, please run the installer script below.
You can [view the source for the installer script](./deployments/installer/install.ps1)
and use it on your hosts in powershell by running:

`& {Set-ExecutionPolicy Bypass -Scope Process -Force; $script = ((New-Object System.Net.WebClient).DownloadString('https://dl.signalfx.com/signalfx-agent.ps1')); $params = @{access_token = "YOUR_SIGNALFX_API_TOKEN"; ingest_url = "https://ingest.YOUR_SIGNALFX_REALM.signalfx.com"; api_url = "https://api.YOUR_SIGNALFX_REALM.signalfx.com"}; Invoke-Command -ScriptBlock ([scriptblock]::Create(". {$script} $(&{$args} @params)"))}`

The agent should be installed to `\Program Files\SignalFx\SignalFxAgent`, and the default configuration file
should be installed at `\ProgramData\SignalFxAgent`.

#### Chef
We offer a Chef cookbook to install and configure the agent.  See [the cookbook
source](./deployments/chef) and [on the Chef
Supermarket](https://supermarket.chef.io/cookbooks/signalfx_agent).

#### Puppet
We also offer a Puppet manifest to install and configure the agent on Linux.  See [the
manifest source](./deployments/puppet) and [on the Puppet
Forge](https://forge.puppet.com/signalfx/signalfx_agent/readme).

#### Ansible
We also offer an Ansible Role to install and configure the Smart Agent on Linux.  See [the
role source](https://github.com/signalfx/signalfx-agent/tree/master/deployments/ansible).

#### Salt
We also offer a Salt Formula to install and configure the Smart Agent on Linux.  See [the
formula source](https://github.com/signalfx/signalfx-agent/tree/master/deployments/salt).

#### Docker Image
See [Docker Deployment](./deployments/docker) for more information.

#### Kubernetes
See our [Kubernetes setup instructions](./docs/kubernetes-setup.md) and the
documentation on [Monitoring
Kubernetes](https://docs.signalfx.com/en/latest/integrations/kubernetes-quickstart.html)
for more information.

#### AWS Elastic Container Service (ECS)
See the [ECS directory](./deployments/ecs), which includes a sample
config and task definition for the agent.

### Bundles
We offer the agent in the following forms:

#### Debian Package
We provide a Debian package repository that you can make use of with the
following commands:

```sh
curl -sSL https://splunk.jfrog.io/splunk/signalfx-agent-deb/splunk-B3CD4420.gpg > /etc/apt/trusted.gpg.d/splunk.gpg
echo 'deb https://splunk.jfrog.io/splunk/signalfx-agent-deb release main' > /etc/apt/sources.list.d/signalfx-agent.list
apt-get update
apt-get install -y signalfx-agent
```

#### RPM Package
We provide a RHEL/RPM package repository that you can make use of with the
following commands:

```sh
cat <<EOH > /etc/yum.repos.d/signalfx-agent.repo
[signalfx-agent]
name=SignalFx Agent Repository
baseurl=https://splunk.jfrog.io/splunk/signalfx-agent-rpm/release
gpgcheck=1
gpgkey=https://splunk.jfrog.io/splunk/signalfx-agent-rpm/splunk-B3CD4420.pub
enabled=1
EOH

yum install -y signalfx-agent
```

#### Linux Standalone tar.gz
If you don't want to use a distro package, we offer a
.tar.gz that can be deployed to the target host.  This bundle is available for
download on the [Github Releases
Page](https://github.com/signalfx/signalfx-agent/releases) for each new
release.

To use the bundle:

1) Unarchive it to a directory of your choice on the target system.

2) Go into the unarchived `signalfx-agent` directory and run
`bin/patch-interpreter $(pwd)`.  This ensures that the binaries in the bundle
have the right loader set on them since your host's loader may not be
compatible.

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

See the section on [Privileges](#privileges) for information on the
capabilities the agent requires.

3) Run the agent by invoking the archive path
`signalfx-agent/bin/signalfx-agent -config <path to config.yaml>`.  By default,
the agent logs only to stdout/err. If you want to persist logs, you must direct
the output to a log file or other log management system.  See the
[signalfx-agent command](./docs/signalfx-agent.1.man) doc for more information on
supported command flags.

#### Windows Standalone .zip
If you don't want to use the installer script, we offer a
.zip that can be deployed to the target host.  This bundle is available for
download on the [Github Releases
Page](https://github.com/signalfx/signalfx-agent/releases) for each new
release.

Before proceeding make sure the following requirements are met.
- [.Net Framework 3.5](https://docs.microsoft.com/en-us/dotnet/framework/install/dotnet-35-windows-10) (Windows 8+)
- [Visual C++ Compiler for Python 2.7](https://www.microsoft.com/EN-US/DOWNLOAD/DETAILS.ASPX?ID=44266)

To use the bundle:

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

See the section on [Privileges](#privileges) for information on the
capabilities the agent requires.

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

### Privileges

#### Linux
When using the [host observer](./docs/observers/host.md), the agent requires
the [Linux
capabilities](http://man7.org/linux/man-pages/man7/capabilities.7.html)
`DAC_READ_SEARCH` and `SYS_PTRACE`, both of which are necessary to allow the
agent to determine which processes are listening on network ports on the host.
Otherwise, there is nothing built into the agent that requires privileges.
When using a package to install the agent, the agent binary is given those
capabilities in the package post-install script, but the agent is run as the
`signalfx-agent` user.  If you are not using the `host` observer, then you can
strip those capabilities from the agent binary if so desired.

You should generally not run the agent as `root` unless you can't use
capabilities for some reason.

#### Windows
On Windows the agent must be installed and run under an administrator account.

## Configuration

The agent is configured primarily from a YAML file. By default, the agent config
is installed at and looked for at `/etc/signalfx/agent.yaml` on Linux and
`\ProgramData\SignalFxAgent\agent.yaml` on Windows. This can be
overridden by the `-config` command line flag.

For the full schema of the config, see [Config Schema](./docs/config-schema.md).

For information on how to configure the agent from remote sources, such as
other files on the filesystem or KV stores such as Etcd, see [Remote
Configuration](./docs/remote-config.md).

## Logging

The default log level is `info`, which will log anything noteworthy in the
agent without spamming the logs too much.  Most of the `info` level logs are on
startup and upon service discovery changes.  `debug` will create very verbose
log output and should only be used when trying to resolve a problem with the
agent.  You can change the log level with the `logging: {level: info}` YAML
config option.

The agent will emit logs in either an unstructed text (default) or JSON format.
You can configure it to emit JSON logs with the `logging: {format: json}` YAML
config option.

### Linux
Currently the agent only supports logging to stdout/stderr, which will
generally be redirected by the init scripts we provide to either a file at
`/var/log/signalfx-agent.log` or to the systemd journal on newer distros.

### Windows
On Windows, the agent will log to the console when executed directly in a shell.
If the agent is configured as a windows service, log events will be logged to the
Windows Event Log.  Use the Event Viewer application to read the logs.  The Event
Viewer is located under `Start > Administrative Tools > Event Viewer`.  You can
see logged events from the agent service under `Windows Logs > Application`.

## Proxy Support

To use an HTTP(S) proxy, set the environment variable `HTTP_PROXY` and/or
`HTTPS_PROXY` in the container configuration to proxy either protocol.  The
SignalFx ingest and API servers both use HTTPS.  If the `NO_PROXY` envvar
exists, the agent will automatically append the local services to the envvar to
not use the proxy.

If the agent is running as a local service on the host, refer to the host's 
service management documentation for how to pass environment variables to the
agent service in order to enable proxy support when the agent service is started.  

For example, if the host services are managed by systemd, create the 
`/etc/systemd/system/signalfx-agent.service.d/myproxy.conf` file and add the 
following to the file:
```
[Service]
Environment="HTTP_PROXY=http://proxy.example.com:8080/"
Environment="HTTPS_PROXY=https://proxy.example.com:8081/"
```
Then execute `systemctl daemon-reload` and `systemctl restart signalfx-agent.service`
to restart the agent service with proxy support.

### Sys-V based init.d systems: Debian * RHEL

Create `/etc/default/signalfx-agent` with the following contents:

```bash
HTTP_PROXY="http://proxy.example.com:8080/"
HTTPS_PROXY="https://proxy.example.com:8081/"
```

## Diagnostics
The agent serves diagnostic information on an HTTP server at the address
configured by the `internalStatusHost` and `internalStatusPort` option.  As a
convenience, the command `signalfx-agent status` will read this server and dump
out its contents.  That command will also explain how to get further diagnostic
information.

Also see our [FAQ](./docs/faq.md) for more troubleshooting help.

## Development

If you wish to contribute to the agent, see the [Developer's
Guide](./docs/development.md).


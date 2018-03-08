# SignalFx Agent (SmartAgent)

[![GoDoc](https://godoc.org/github.com/signalfx/signalfx-agent?status.svg)](https://godoc.org/github.com/signalfx/signalfx-agent)

The SignalFx Agent (a.k.a. the "SmartAgent") is a metric agent written in Go
for monitoring nodes and application services in a variety of different
environments.

## Concepts

The agent has three main components:

1) _Observers_ that discover applications and services running on the host
2) _Monitors_ that collect metrics from the host and applications
3) The _Writer_ that sends the metrics collected by monitors to SignalFx.

### Observers

Observers are what watch the various environments we support to discover running
services and automatically configure the agent to send metrics for those
services.

See [Observer Config](./docs/observer-config.md) for a list of supported
observers and their configuration.

### Monitors

Monitors are what collect metrics from the host system and services.  They are
configured under the `monitors` list in the agent config.  For monitors of
applications, you can configure a discovery rule on the monitor so that a
separate instance of the monitor is created for each discovered instance of
applications that match that discovery rule.  See [Auto
Discovery](./docs/auto-discovery.md) for more information.

Many of the monitors rely on a third-party "super monitor",
[collectd](https://collectd.org), under the covers to do a lot of the metric
collection, although we also have monitors apart from Collectd.  They are
configured in the same way, however.

See [Monitor Config](./docs/monitor-config.md) for a list of supported monitors
and their configuration.

The agent is primarily intended to monitor services/applications running on the
same host as the agent.  This is in keeping with the collectd model.  The main
issue with monitoring services on other hosts is that the `host` dimension that
collectd sets on all metrics will currently get set to the hostname of the
machine that the agent is running on.  This allows everything to have a
consistent `host` dimension so that metrics can be matched to a specific
machine during metric analysis.

## Configuration

The agent is configured primarily from a YAML file (by default,
`/etc/signalfx/agent.yaml`, but this can be overridden by the `-config` command
line flag).  

For the full schema of the config, see [Config Schema](./docs/config-schema.md).

See [Remote Configuration](./docs/remote-config.md) for information on how to
configure the agent from remote sources such as other files on the filesystem
or KV stores such as Etcd.

## Installation

The agent is available in both a containerized and standalone form for Linux.
Whatever form you use, the dependencies are completely bundled along with the
agent, including a Java JRE runtime and a Python runtime, so there are no
additional dependencies required.  This means that the agent should work on any
relatively modern Linux distribution (kernel version 2.6+).

### Bundles
We offer the agent in the following forms:

#### Docker Image
The agent is available as a Docker image at
[quay.io/signalfx/signalfx-agent](https://quay.io/signalfx/signalfx-agent). The
image is tagged using the same agent version scheme.

#### Debian Package
We provide a Debian package repository that you can make use of with the
following commands:

```sh
curl -sSL https://dl.signalfx.com/debian.gpg > /etc/apt/trusted.gpg.d/signalfx.gpg
echo 'deb https://dl.signalfx.com/debs/signalfx-agent/main /' > /etc/apt/sources.list.d/signalfx-agent.list
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
baseurl=https://dl.signalfx.com/rpms/signalfx-agent/main
gpgcheck=1
gpgkey=https://dl.signalfx.com/yum-rpm.key
enabled=1
EOH

yum install -y signalfx-agent
```

#### Standalone tar.gz
If you don't want to use a distro package, we offer the agent bundle as a
.tar.gz that can be deployed to the target host.  This bundle is available for
download on the [Github Releases
Page](https://github.com/signalfx/signalfx-agent/releases) for each new
release.

To use the bundle:

1) Unarchive it to any directory on the target system you like.

2) Ensure a valid configuration file is available somewhere on the target
system.  The main thing that the distro packages provide -- but that you will
have to provide manually with the bundle -- is a run directory for the agent to
use.  There are three config options that you will especially want to consider
since you aren't installing from a package:

 - `diagnosticsSocketPath` - This is the full path to a UNIX domain socket that
	 the agent will listen on so that the `signalfx-agent status` command can
	 read diagnostic information from a running agent.  It can be blank if you
	 don't desire that functionality.

 - `internalMetricsSocketPath` - This is the full path to a UNIX domain socket
	 that the agent will listen on for requests from the
	 [internal-metrics](./docs/monitors/internal-metrics.md) monitor to gather
	 internal metrics about the agent.  It can also be blank to disable this
	 socket.

 - `collectd.configDir` - This is where the agent writes the managed collectd
	 config since collectd can only be configured by files.  Note that this
	 entire dir will be **wiped by the agent upon startup** so that it doesn't
	 pick up stale collectd config, so be sure that it is not used for anything
	 else.  Also note that these files could have **sensitive information in
	 them** if you have passwords configured for collectd monitors, so you
	 might want to place this dir on a `tmpfs` mount to avoid credentials being
	 persisted on disk.

See the section on [Privileges](#privileges) for information on what
capabilities the agent needs.

3) Run the agent by invoking the archive path
`signalfx-agent/bin/signalfx-agent -config <path to config.yaml>`.  The agent
logs only to stdout/err so it is up to you to direct that to a log file or
other log management system if you wish to persist logs.  See the
[signalfx-agent command](./docs/signalfx-agent.1.md) doc for more information on
supported command flags.

### Deployment Tools
We support the following deployment/configuration management tools to automate the
installation process:

#### Installer Script
For non-containerized environments, there is a convenience script that you can
run on your host to install the agent package.  This is useful for testing and
trails, but for full-scale deployments you will probably want to use a
configuration management system like Chef or Puppet.  You can [view the source
for the installer script](./packaging/installer/install.sh) and use it on your
hosts by running:

```sh
curl -sSL https://dl.signalfx.com/signalfx-agent.sh > /tmp/signalfx-agent.sh
sh /tmp/signalfx-agent.sh <access token>
```

#### Chef
We offer a Chef cookbook to install and configure the agent.  See [the cookbook
source](./deployments/chef) and INSERT URL TO SUPERMARKET.

#### Puppet
We also offer a Puppet manifest to install and configure the agent.  See [the
manifest source](./deployments/puppet) and INSERT THE PUPPET FORGE LINK.

#### Kubernetes
See our [Kubernetes Quickstart
Guide](https://docs.signalfx.com/en/latest/integrations/kubernetes-quickstart.html)
for more information.

### Privileges

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

## Logging
Currently the agent only supports logging to stdout/stderr, which will
generally be redirected by the init scripts we provide to either a file at
`/var/log/signalfx-agent.log` or to the systemd journal on newer distros. The
default log level is `info`, which will log anything noteworthy in the agent
without spamming the logs too much.  Most of the `info` level logs are on
startup and upon service discovery changes.  `debug` will create very verbose
log output and should only be used when trying to resolve a problem with the
agent.

## Proxy Support

To use a HTTP(S) proxy, set the environment variable `HTTP_PROXY` and/or
`HTTPS_PROXY` in the container configuration to proxy either protocol.  The
SignalFx ingest and API servers both use HTTPS.  The agent will automatically
manipulate the `NO_PROXY` envvar to not use the proxy for local services.

## Diagnostics
The agent serves diagnostic information on a UNIX domain socket at the path
configured by the `diagnosticsSocketPath` option.  The socket takes no input,
but simply dumps it's current status back upon connection.  As a convenience,
the command `signalfx-agent status` will read this socket and dump out its
contents.

## Development

If you wish to contribute to the agent, see the [Developer's
Guide](./docs/development.md).


# SignalFx Smart Agent 

[![GoDoc](https://godoc.org/github.com/signalfx/signalfx-agent?status.svg)](https://godoc.org/github.com/signalfx/signalfx-agent)
[![CircleCI](https://circleci.com/gh/signalfx/signalfx-agent.svg?style=shield)](https://circleci.com/gh/signalfx/signalfx-agent)

The SignalFx Smart Agent is a metric agent written in Go for monitoring
infrastructure and application services in a variety of different environments.
It is meant as a successor to our previous [collectd
agent](https://github.com/signalfx/collectd), but still uses that internally --
so any existing Python or C-based collectd plugins will still work without
modification.

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
2) _Monitors_ that collect metrics from the host and applications
3) The _Writer_ that sends the metrics collected by monitors to SignalFx.

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
to SignalFx on a regular basis.  By default it batches metrics for 1 second
before sending.  There are a few things that can be
[configured](./docs/config-schema.md#writer) in the writer, but this is generally
unnecessary.

## Installation

The agent is available for Linux in both a containerized and standalone form.
Whatever form you use, the dependencies are completely bundled along with the
agent, including a Java JRE runtime and a Python runtime, so there are no
additional dependencies required.  This means that the agent should work on any
relatively modern Linux distribution (kernel version 2.6+).

### Deployment
We support the following deployment/configuration management tools to automate the
installation process.  See [Bundles](#bundles) for a list of underlying
packages for the agent.

#### Installer Script
For non-containerized environments, there is a convenience script that you can
run on your host to install the agent package.  This is useful for testing and
trials, but for full-scale deployments you will probably want to use a
configuration management system like Chef or Puppet.  You can [view the source
for the installer
script](./deployments/installer/install.sh)
and use it on your hosts by running:

```sh
curl -sSL https://dl.signalfx.com/signalfx-agent.sh > /tmp/signalfx-agent.sh
sudo sh /tmp/signalfx-agent.sh YOUR_SIGNALFX_API_TOKEN
```

#### Chef
We offer a Chef cookbook to install and configure the agent.  See [the cookbook
source](./deployments/chef) and [on the Chef
Supermarket](https://supermarket.chef.io/cookbooks/signalfx_agent).

#### Puppet
We also offer a Puppet manifest to install and configure the agent.  See [the
manifest source](./deployments/puppet) and [on the Puppet
Forge](https://forge.puppet.com/signalfx/signalfx_agent/readme).

#### Ansible
We also offer an Ansible Role to install and configure the Smart Agent.  See [the
role source](https://github.com/signalfx/signalfx-agent/tree/master/deployments/ansible).

#### Salt
We also offer a Salt Formula to install and configure the Smart Agent.  See [the
formula source](https://github.com/signalfx/signalfx-agent/tree/master/deployments/salt).

#### Kubernetes
See our [Kubernetes setup instructions](./docs/kubernetes-setup.md) and the
documentation on [Monitoring
Kubernetes](https://docs.signalfx.com/en/latest/integrations/kubernetes-quickstart.html)
for more information.

### Bundles
We offer the agent in the following forms:

#### Docker Image
We provide a Docker image at
[quay.io/signalfx/signalfx-agent](https://quay.io/signalfx/signalfx-agent). The
image is tagged using the same agent version scheme.

If you are using Docker outside of Kubernetes, you can run the agent in a
Docker container and still gather metrics on the underlying host by running it
with the following flags:

```sh
$ docker run \
    --name signalfx-agent \
    --pid host \
    --net host \
    -v /:/hostfs:ro \
    -v /var/run/docker.sock:/var/run/docker.sock:ro \
    -v /etc/signalfx/:/etc/signalfx/:ro \
    quay.io/signalfx/signalfx-agent:<version>
```

This assumes you have the agent config in the conventional directory
(`/etc/signalfx`) on the root mount namespace.

If you have the Docker API available through the conventional UNIX domain
socket, you should mount that in to be able to use the
[docker-container-stats](./docs/monitors/docker-container-stats.md) monitor.

It is necessary to mount in the host root filesystem at `/hostfs` in order to
get disk usage metrics for the host filesystems using the
[collectd/df](./docs/monitors/collectd-df.md).  You will need to set the
`hostFSPath: /hostfs` config option on that monitor to make it use this
non-default path.

The only other special config you will need is the `etcPath: /hostfs/etc`
option under the
[collectd/signalfx-metadata](./docs/monitors/collectd-signalfx-metadata.md)
monitor config.  This tells it where to find certain files like
`/etc/os-release` that are used to generate host metadata such as the Linux
distro and version.

You may also want to use the [Docker observer](./docs/observers/docker.md) to
automatically discover other containers running in the same Docker engine.

#### Debian Package
We provide a Debian package repository that you can make use of with the
following commands:

```sh
curl -sSL https://dl.signalfx.com/debian.gpg > /etc/apt/trusted.gpg.d/signalfx.gpg
echo 'deb https://dl.signalfx.com/debs/signalfx-agent/final /' > /etc/apt/sources.list.d/signalfx-agent.list
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
baseurl=https://dl.signalfx.com/rpms/signalfx-agent/final
gpgcheck=1
gpgkey=https://dl.signalfx.com/yum-rpm.key
enabled=1
EOH

yum install -y signalfx-agent
```

#### Standalone tar.gz
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
`signalfx-agent/bin/signalfx-agent -config <path to config.yaml>`.  The agent
logs only to stdout/err so it is up to you to direct that to a log file or
other log management system if you wish to persist logs.  See the
[signalfx-agent command](./docs/signalfx-agent.1.md) doc for more information on
supported command flags.

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

## Configuration

The agent is configured primarily from a YAML file (by default,
`/etc/signalfx/agent.yaml`, but this can be overridden by the `-config` command
line flag).  

For the full schema of the config, see [Config Schema](./docs/config-schema.md).

For information on how to configure the agent from remote sources, such as
other files on the filesystem or KV stores such as Etcd, see [Remote
Configuration](./docs/remote-config.md).

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

To use an HTTP(S) proxy, set the environment variable `HTTP_PROXY` and/or
`HTTPS_PROXY` in the container configuration to proxy either protocol.  The
SignalFx ingest and API servers both use HTTPS.  The agent will automatically
manipulate the `NO_PROXY` envvar to not use the proxy for local services.

## Diagnostics
The agent serves diagnostic information on a UNIX domain socket at the path
configured by the `diagnosticsSocketPath` option.  The socket takes no input,
but simply dumps its current status back upon connection.  As a convenience,
the command `signalfx-agent status` will read this socket and dump out its
contents.

The agent status output has the following sections:

 - **Version**: The agent version and build time
 - **Agent Configuration**: The current configuration in use by the agent, with
	 secret values replaced by `*`s.  Default values will be shown here if they
	 were not set in the agent config file.
 - **Writer Status**: The status and metrics about the writer component which
	 writes datapoints to SignalFx.
 - **Observers**: The active observers in the agent
 - **Monitor Configurations (Not necessarily active)**: A list of monitor
	 configurations that are in place.  If a configuration has a discovery rule
	 but no discovered endpoints match that rule, there will not be any active
	 instances of this monitor.
 - **Active Monitors**: Monitors instances that are actively monitoring
	 something.  There may be multiple instances of these per configuration
	 above if there is a discovery rule that matches multiple services.
 - **Discovered Endpoints**: A list of the endpoints discovered by the agent's
	 observers.  The fields shown there will be the fields used when matching
	 discovery rules to a discovered endpoint.
 - **Bad Monitor Configurations**: This will be a set of monitor configurations
	 that did not validate and the associated error.  Bad monitor configuration
	 generally does not prevent the agent from starting up, but will prevent
	 that monitor from ever instantiating.

Also see our [FAQ](./docs/faq.md) for more troubleshooting help.

## Development

If you wish to contribute to the agent, see the [Developer's
Guide](./docs/development.md).


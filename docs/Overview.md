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

 - [Components](#components)
 - [Configuration](#configuration)
 - [Logging](#logging)
 - [Proxy Support](#proxy-support)
 - [Diagnostics](#diagnostics)
 - [Development](#development)

## Components

The agent has three main components:

1) _Observers_ that discover applications and services running on the host
2) _Monitors_ that collect metrics from the host and applications
3) The _Writer_ that sends the metrics collected by monitors to SignalFx.

### Observers

Observers watch the various environments that we support to discover running
services and automatically configure the agent to send metrics for those
services.

For a list of supported observers and their configurations,
see [Observer Config](./observer-config.md).

### Monitors

Monitors collect metrics from the host system and services.  They are
configured under the `monitors` list in the agent config.  For
application-specific monitors, you can define discovery rules in your monitor
configuration. A separate monitor instance is created for each discovered
instance of applications that match a discovery rule. See [Auto
Discovery](./doc/auto-discovery.md) for more information.

Many of the monitors are built around [collectd](https://collectd.org), an open
source third-party monitor, and use it to collect metrics. Some other monitors
do not use collectd. However, either type is configured in the same way.

For a list of supported monitors and their configurations, 
see [Monitor Config](./doc/monitor-config.md).

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
[configured](./doc/config-schema.md#writer) in the writer, but this is generally
only necessary if you have a very large number of metrics flowing through a
single agent.

## Configuration

The agent is configured primarily from a YAML file. By default, the agent config
is installed at and looked for at `/etc/signalfx/agent.yaml` on Linux and
`\ProgramData\SignalFxAgent\agent.yaml` on Windows. This can be
overridden by the `-config` command line flag.  

For the full schema of the config, see [Config Schema](./doc/config-schema.md).

For information on how to configure the agent from remote sources, such as
other files on the filesystem or KV stores such as Etcd, see [Remote
Configuration](/remote-config.md).

## Logging

### Linux
Currently the agent only supports logging to stdout/stderr, which will
generally be redirected by the init scripts we provide to either a file at
`/var/log/signalfx-agent.log` or to the systemd journal on newer distros. The
default log level is `info`, which will log anything noteworthy in the agent
without spamming the logs too much.  Most of the `info` level logs are on
startup and upon service discovery changes.  `debug` will create very verbose
log output and should only be used when trying to resolve a problem with the
agent.

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

## Diagnostics
The agent serves diagnostic information on an HTTP server at the address
configured by the `internalStatusHost` and `internalStatusPort` option.  As a
convenience, the command `signalfx-agent status` will read this server and dump
out its contents.  That command will also explain how to get further diagnostic
information.

Also see our [FAQ](./doc/faq.md) for more troubleshooting help.

## Development

If you wish to contribute to the agent, see the [Developer's
Guide](./doc/development.md).


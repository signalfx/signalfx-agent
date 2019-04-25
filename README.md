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
 - [Installation](#installation)

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

To get started installing the Smart Agent on a single host, see 
[Smart Agent Quick Install](./docs/smart-agent-quick-install.md).

To install Smart Agent on multiple hosts using bundles or packages, see [Smart Agent Next Steps](./docs/smart-agent_next_steps.md)



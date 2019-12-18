# Writing a Monitor

Monitors go out to the environment around the agent and collect
metrics about running services or platforms.  Adding a new monitor is
relatively simple.

We considered using the new Go 1.8+ plugin architecture to implement plugins
but have currently decided against it due to various issues still outstanding
with the plugin framework that make the benefits relatively minimal for a great
deal of added complexity and artifact size.  Therefore right now, new monitors
must be compiled into the agent binary.

First, create a new package within the `github.com/signalfx/signalfx-agent/pkg/monitors`
package (or inside the `pkg/monitors/collectd` package if creating a
collectd wrapper monitor, see below for more on collectd monitors).  Inside
that package create a single module named whatever you like that will hold the
monitor code. If your monitor gets complicated, you can of course split it up
into multiple modules or even packages as desired.

Here is a minimalistic example of a monitor:

```go
package mymonitor

import (
	"time"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	log "github.com/sirupsen/logrus"
)

func init() {
	monitors.RegisterWithMetadata(&monitorMetadata,
		func() interface{} { return &Monitor{} },
		&Config{})
}

// Config for monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

	// Required for monitors that accept auto-discovered endpoints
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
	Name string `yaml:"name"`

	// Holds my config string
	MyVar string `yaml:"myVar"`
}

// Validate will check the config for correctness.  This method is optional.
func (c *Config) Validate() error {
	if c.MyVar == "" {
		return errors.New("myVar is required")
	}
	return nil
}

// Monitor that collectd metrics.
type Monitor struct {
	// This will be automatically injected to the monitor instance before
	// Configure is called.
	Output types.Output
	cancel func()
}

// Configure and kick off internal metric collection
func (m *Monitor) Configure(conf *Config) error {
	// Start the metric gathering process here.
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())
	utils.RunOnInterval(ctx, func() {

		// This would be a more complicated in a real monitor, but this
		// shows the basic idea of using the Output interface to send
		// datapoints.
		m.Output.SendDatapoints([]*datapoint.Datapoint{
			datapoint.New("my-monitor.requests", map[string]string{"env": "test"}, 100, datapoint.Gauge, time.Now())
		})

	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown the monitor
func (m *Monitor) Shutdown() {
	// Stop any long-running go routines here
	if m.cancel != nil {
		m.cancel()
	}
}
```

There are two data types that are essential to a monitor: the configuration and the
monitor itself.  By convention these are called `Config` and `Monitor` but the
names don't matter and can be anything you like.

## Configuration

The config struct is where any configuration of your monitor will go.  It must
embed the
`github.com/signalfx/signalfx-agent/pkg/core/config.MonitorConfig` struct,
which includes generic configuration common to all monitors.  Configuration of
the agent (and also monitors) is driven by YAML, and it is best practice to
explicitly state the YAML key for your config values instead of letting the
YAML interpreter derive it by default.  See [the golang YAML
docs](https://godoc.org/gopkg.in/yaml.v2) for more information.

Configuration fields can be of any type so long as it can be deserialized from
YAML.

By default, the agent will ensure any provided configuration matches the types
specified in your config struct.  If you want more advanced validation, you can
implement the `Validate() error` method on your config type.  This will be
called by the agent with the config struct fully populated with the provided
config before calling the `Configure` method of your monitor.  If it returns a
non-nil error, the error will be logged and the `Configure` method will not be
called.

The embedded `MonitorConfig` struct contains a field called `IntervalSeconds`.
Your monitor must make a good effort to send metrics at this interval, but
nothing is enforcing it so you have total freedom to follow the interval
however you like.  There is nothing in the agent that calls your monitor on a
regular interval.

Monitor config is considered immutable once configured.  That means that your
monitor's `Configure` method will never be called more than once for a given
monitor instance.  You are free to mutate the config instance within the
monitor code however, if desired.

### Auto Discovery

If your monitor is watching service endpoints that are appropriate for auto
discovery (e.g. a web service), you have to tell the agent this by specifying
the `acceptsEndpoints:"true"` tag on the embedded `MonitorConfig` struct in
your config struct type.  See the example above for what this looks like.
Then, you must specify three YAML fields in your config struct that all
discovered service endpoints provide in their configuration data:

- `host` (string): The hostname or IP address of the discovered service
- `port` (uint16): The port number of the service (can be TCP or UDP)
- `name` (string): A human-friendly name for the service as determined by the
observer that generated the endpoint.

These are normally called `Host`, `Port` and `Name`, but you can call them
whatever you like as long as the YAML name is correct.

You should also specify the `yaml:",inline"` tag of the embedded
`MonitorConfig` field so that observers that create endpoints with config that
overrides fields in that embedded struct can be correctly merged into the
config struct (e.g. the Kubernetes observer can set the interval via
annotations).

When an endpoint is discovered by an observer, the observer sets configuration
on the endpoint that then gets merged into your monitor's config before
`Configure` is called.  As far as your monitor's `Configure` method is
concerned, there is no difference between an auto-discovered endpoint and a
manually specified one.

## Monitor Struct

Every monitor must have a struct type that defines it.  This is what gets
instantiated by the agent and what has the `Configure` method that gets called
after being instantiated.  A new instance of your monitor struct will be
created for each distinct configuration, so `Configure()` will only be called
once per monitor instance.

A monitor's interface is simple: there must be a `Configure` method and there
can optionally be a `Shutdown` method.  The `Configure` method must take a
pointer to the same config struct type registered for the monitor (see below
for registration).

There is a special field that can be specified by the monitor struct that will
be automatically populated by the agent:

- `Output "github.com/signalfx/signalfx-agent/pkg/monitors/types".Output`: This is what
    is used to send data from the monitor back to the agent, and then on to
    SignalFx.  This value has three methods:

    - `SendDatapoints([]*"github.com/signalfx/golib/v3/datapoint".Datapoint)`:
		Sends a set of datapoints, appending any extra dimensions specified in
		the configuration or by the service endpoint associated with the
		monitor.

    - `SendEvent(*"github.com/signalfx/golib/v3/event".Event)`: Sends an event.

	- `SendDimensionUpdate(*"github.com/signalfx/signalfx-agent/pkg/monitors/types".Dimension)`:
		Sends property updates for a specific dimension key/value pair.

The name and type of the struct field must be exactly as specified or else it
will not be injected.

## Registration

The [init function](https://golang.org/doc/effective_go.html#init) of your
package must register your monitor with the agent core.  This is done by
calling the `Register` function in the `monitors` package.  This function takes
three arguments:

1) The type of the monitor.  This is a string that should be dash delimited.
You will use this type in the agent configuration to identify the monitor.

2) A niladic factory function that returns a new uninitialized instance of your
monitor.

3) A reference to an uninitialized instance of your monitor's config struct.
This is used to perform config validation in the agent core, as well as to pass
the right type to the Configure method of the monitor.

The `Configure` method will receive a reference to the config struct that you
registered with the agent.  It is guaranteed to have passed its `Validate`
method, if provided.

## Create Dependency From Agent Core

To force the agent to compile and statically link in your new monitor code in
the binary, you must include the package in the
`github.com/signalfx/signalfx-agent/pkg/core/modules.go` module.

## Shutdown

Most monitors will need to do some kind of shutdown logic to avoid leaking
memory/goroutines.  This should be done in the `Shutdown()` method if your
monitor.  The agent will call this method if provided when the monitor is no
longer needed.  It should not block.

The `Shutdown()` method will not be called more than once.

If your monitor's configuration is changed in the agent, the agent will
shutdown existing monitors dependent on that config and recreate them with the
new config.

Note that `Shutdown` is **not** called if `Configure` returns an error, so the
`Configure` method should clean up anything it may have started before
returning the error.

## Documentation

To make your monitor show up in the auto-generated docs, you should create a
`metadata.yaml` file in the same package as your monitor. See an existing
monitor as an example. The `monitorType` property should match exactly the type
that you register your monitor as.

You should also document all metrics that your monitor emits. The metric types
you can specify are `gauge`, `counter`, `cumulative`, and `timestamp`.

You should also document any monitor-specific dimensions that your monitor
attaches to datapoints that it emits.

## Best Practices

 - It is best to send metrics immediately upon a monitor being configured and
   then at the specified interval so that metrics start coming out of the
   agent as soon as possible.  This will help minimize the chance of metric
   gaps.

## Collectd-based Monitors

Collectd runs as a subprocess of the agent.  It's configuration is managed
entirely by the agent.  Every collectd plugin that sends metrics is
encapsulated in a monitor that manages the collectd plugin config, as well as
making sure collectd is restarted when its config is added or changes.

The simplest way to get started is to simply copy an existing collectd monitor
package and alter it.  Much of the package code is boilerplate anyway, although
even that has been kept to a minimum.  The main things you will need to change
is the config template and the config struct.

Almost all of the work involved in configuring and restarting collectd is
encapsulated in a type called `MonitorCore` that resides in the
`github.com/signalfx/signalfx-agent/pkg/monitors/collectd` package.  The monitor must
only embed this type in the main monitor struct and make sure it gets instantiated
with a collectd config template in the monitor factory function.

The convention for collectd monitors is to write the collectd config template
in a standalone plain text file and use Go's code generation feature to render
it into a Go module.  Just copy the `go generate` comment from another collectd
monitor and change the name of the template file to match yours.  Collectd
config templates use [Go templating](https://golang.org/pkg/text/template/).

The same things about endpoints and config apply to the collectd monitors as
above.  Collectd monitors are not treated any differently from the agent's
perspective.

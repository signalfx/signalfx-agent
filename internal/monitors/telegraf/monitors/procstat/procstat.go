package procstat

import (
	"context"
	"time"

	"github.com/ulule/deepcopier"

	telegrafInputs "github.com/influxdata/telegraf/plugins/inputs"
	telegrafPlugin "github.com/influxdata/telegraf/plugins/inputs/procstat"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/baseemitter"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

const monitorType = "telegraf/procstat"

// MONITOR(telegraf/procstat): This monitor reports metrics about processes.
// This monitor is based on the Telegraf procstat plugin.  More information about the Telegraf plugin
// can be found [here](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/procstat).
//
// Please note that the Smart Agent only supports the `native` pid finder and the options
// `cgroup` and `systemd unit` are not supported at this time.
//
// On Linux hosts, this monitor relies on the `/proc` filesystem.
// If the underlying host's `/proc` file system is mounted somewhere other than
// /proc please specify the path using the top level configuration `procPath`.
//
// Sample Yaml Configuration
//
// ```yaml
// procPath: /proc
// monitors:
//  - type: telegraf/procstat
//    exe: "signalfx-agent*"
// ```
//

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"false"`
	// The name of an executable to monitor.  (ie: `exe: "signalfx-agent*"`)
	Exe string `yaml:"exe"`
	// Pattern to match against.  On Windows the pattern should be in the form of a WMI query.
	// (ie: `pattern: "%signalfx-agent%"`)
	Pattern string `yaml:"pattern"`
	// Username to match against
	User string `yaml:"user"`
	// Path to Pid file to monitor.  (ie: `pidFile: "/var/run/signalfx-agent.pid"`)
	PidFile string `yaml:"pidFile"`
	// Used to override the process name dimension
	ProcessName string `yaml:"processName"`
	// Prefix to be added to each dimension
	Prefix string `yaml:"prefix"`
	// Whether to add PID as a dimension instead of part of the metric name
	PidTag bool `yaml:"pidTag"`
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
}

// fetch the factory used to generate the perf counter plugin
var factory = telegrafInputs.Inputs["procstat"]

// Configure the monitor and kick off metric syncing
func (m *Monitor) Configure(conf *Config) (err error) {
	plugin := factory().(*telegrafPlugin.Procstat)

	// create the accumulator
	ac := accumulator.NewAccumulator(baseemitter.NewEmitter(m.Output, logger))

	// copy configurations to the plugin
	if err = deepcopier.Copy(conf).To(plugin); err != nil {
		logger.Error("unable to copy configurations to plugin")
		return err
	}

	// set the pid finder to native because we don't bundle pgrep at the moment
	// and containerizing pgrep is likely difficult
	plugin.PidFinder = "native"

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		if err := plugin.Gather(ac); err != nil {
			logger.Error(err)
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return err
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

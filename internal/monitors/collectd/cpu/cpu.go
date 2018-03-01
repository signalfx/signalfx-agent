package cpu

//go:generate collectd-template-to-go cpu.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/cpu"

// MONITOR(collectd/cpu): This monitor collects cpu usage data using the
// collectd `cpu` plugin.  It aggregates the per-core CPU data into a single
// metric and sends it to the SignalFx Metadata plugin in collectd, where the
// raw jiffy counts from the `cpu` plugin are converted to percent utilization
// (the `cpu.utilization` metric).
//
// See https://collectd.org/wiki/index.php/Plugin:CPU

// GAUGE(cpu.utilization): Percentage of total CPU used within the last metric
// interval cycle.

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			MonitorCore: *collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `singleInstance:"true"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	return m.SetConfigurationAndRun(conf)
}

// +build !windows

package cpufreq

//go:generate collectd-template-to-go cpufreq.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/cpufreq"

// MONITOR(collectd/cpufreq): Monitors the actual clock speed of each CPU on a
// host.  Useful for systems that vary the clock speed to conserve energy.
//
// See https://collectd.org/wiki/index.php/Plugin:CPUFreq

// GAUGE(cpufreq.<N>): The processor frequency in Hertz for the <N>th processor
// on the system.

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

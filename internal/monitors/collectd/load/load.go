// +build !windows

package load

//go:generate collectd-template-to-go load.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

// MONITOR(collectd/load): Monitors process load on the host using the collectd
// [Load plugin](https://collectd.org/wiki/index.php/Plugin:Load).

// GAUGE(load.longterm): Average CPU load per core over the last 15 minutes
// GAUGE(load.midterm): Average CPU load per core over the last five minutes
// GAUGE(load.shortterm): Average CPU load per core over the last one minute

const monitorType = "collectd/load"

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

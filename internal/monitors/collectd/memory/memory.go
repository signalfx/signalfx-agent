// +build !windows

package memory

//go:generate collectd-template-to-go memory.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/memory"

// MONITOR(collectd/memory): Sends memory usage stats for the underlying host.
// See https://collectd.org/wiki/index.php/Plugin:Memory

// GAUGE(memory.buffered): Bytes of memory used for buffering I/O

// GAUGE(memory.cached): Bytes of memory used for disk caching

// GAUGE(memory.free): Bytes of memory available for use

// GAUGE(memory.slab_recl): Bytes of memory, used for SLAB-allocation of kernel
// objects, that can be reclaimed.

// GAUGE(memory.slab_unrecl): Bytes of memory, used for SLAB-allocation of
// kernel objects, that can't be reclaimed

// GAUGE(memory.used): Bytes of memory in use by the system.

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

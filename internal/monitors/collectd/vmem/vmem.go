// +build !windows

package vmem

//go:generate collectd-template-to-go vmem.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/vmem"

// MONITOR(collectd/vmem): Collects information about the virtual memory
// subsystem of the kernel using the [collectd vmem
// plugin](https://collectd.org/wiki/index.php/Plugin:vmem).  There is no
// configuration available for this plugin.

// CUMULATIVE(vmpage_faults.majflt): Number of major page faults on the system
// CUMULATIVE(vmpage_faults.minflt): Number of minor page faults on the system
// CUMULATIVE(vmpage_io.memory.in): Page Ins for Memory
// CUMULATIVE(vmpage_io.memory.out): Page Outs for Memory
// CUMULATIVE(vmpage_io.swap.in): Page Ins for Swap
// CUMULATIVE(vmpage_io.swap.out): Page Outs for Swap
// CUMULATIVE(vmpage_number.free_pages): Number of free memory pages
// CUMULATIVE(vmpage_number.mapped): Number of mapped pages

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

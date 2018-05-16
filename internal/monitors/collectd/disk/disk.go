package disk

//go:generate collectd-template-to-go disk.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/disk"

// MONITOR(collectd/disk): This monitor collects information about the usage of
// physical disks and logical disks (partitions).
//
// See https://collectd.org/wiki/index.php/Plugin:Disk.

// CUMULATIVE(disk_merged.read): The number of disk reads merged into single physical disk access operations.
// CUMULATIVE(disk_merged.write): The number of disk writes merged into single physical disk access operations.
// CUMULATIVE(disk_octets.read): The number of bytes (octets) read from a disk.
// CUMULATIVE(disk_octets.write): The number of bytes (octets) written to a disk.
// CUMULATIVE(disk_ops.read): The number of disk read operations.
// CUMULATIVE(disk_ops.write): The number of disk write operations.
// CUMULATIVE(disk_time.read): The average amount of time it took to do a read operation.
// CUMULATIVE(disk_time.write): The average amount of time it took to do a write operation.

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

	// Which devices to include/exclude
	Disks []string `yaml:"disks" default:"[\"/^loop\\\\d+$/\", \"/^dm-\\\\d+$/\"]"`

	// If true, the disks selected by `disks` will be excluded and all others
	// included.
	IgnoreSelected bool `yaml:"ignoreSelected" default:"true"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	return m.SetConfigurationAndRun(conf)
}

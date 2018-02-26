package disk

//go:generate collectd-template-to-go disk.tmpl

import (
	"github.com/creasty/defaults"
	"github.com/pkg/errors"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/disk"

// MONITOR(collectd/disk): This monitor collects information about the usage of
// physical disks and logical disks (partitions).
//
// See https://collectd.org/wiki/index.php/Plugin:Disk.

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
	if err := defaults.Set(conf); err != nil {
		return errors.Wrap(err, "Could not set defaults for disk monitor")
	}
	return m.SetConfigurationAndRun(conf)
}

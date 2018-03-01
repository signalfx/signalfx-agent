package metadata

//go:generate collectd-template-to-go metadata.tmpl

import (
	"github.com/creasty/defaults"
	"github.com/pkg/errors"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/signalfx-metadata"

// MONITOR(collectd/signalfx-metadata): Collectd Python plugin that aggregates
// various metrics from other collectd plugins.  It also sends host metadata to
// SignalFx through specially formatted events, and sends active process
// ("top") lists on a periodic basis.
//
// See [Python plugin code](https://github.com/signalfx/collectd-signalfx/) and
// [Integrations docs](https://github.com/signalfx/integrations/tree/master/signalfx-metadata).

// GAUGE(cpu.utilization): Percent of CPU used on this host.
// GAUGE(cpu.utilization_per_core): Percent of CPU used on each core.
// GAUGE(disk.summary_utilization): Percent of disk space utilized on all volumes on this host.
// GAUGE(disk.utilization): Percent of disk used on this volume.
// CUMULATIVE(disk_ops.total): Total number of disk read and write operations on this host.
// GAUGE(memory.utilization): Percent of memory in use on this host.
// CUMULATIVE(network.total): Total amount of inbound and outbound network traffic on this host, in bytes.

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
	WriteServerURL       string `yaml:"writeServerURL"`
	// The path to the proc filesystem. Useful to override in containerized
	// environments.
	ProcFSPath string `yaml:"procFSPath" default:"/proc"`
	// The path to the main host config dir. Userful to override in
	// containerized environments.
	EtcPath string `yaml:"etcPath" default:"/etc"`
	// A directory where the metadata plugin can persist the history of
	// successful host metadata syncs so that host metadata is not sent
	// redundantly.
	PersistencePath string `yaml:"persistencePath" default:"/var/lib/misc"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	if err := defaults.Set(conf); err != nil {
		return errors.Wrap(err, "Could not set defaults for signalfx-metadata monitor")
	}

	conf.WriteServerURL = collectd.Instance().WriteServerURL()

	return m.SetConfigurationAndRun(conf)
}

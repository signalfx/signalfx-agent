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
	ProcFSPath           string `yaml:"procFSPath" default:"/proc"`
	EtcPath              string `yaml:"etcPath" default:"/etc"`
	PersistencePath      string `yaml:"persistencePath" default:"/var/lib/misc"`
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

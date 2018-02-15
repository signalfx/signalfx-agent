package processes

//go:generate collectd-template-to-go processes.tmpl

import (
	"github.com/creasty/defaults"
	"github.com/pkg/errors"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/processes"

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
	Processes            []string          `yaml:"processes"`
	ProcessMatch         map[string]string `yaml:"processMatch"`
	CollectContextSwitch bool              `yaml:"collectContextSwitch" default:"false"`
	ProcFSPath           string            `yaml:"procFSPath" default:"/proc"`
}

// Validate will check the config for correctness.
func (c *Config) Validate() error {
	return nil
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) error {
	if err := defaults.Set(conf); err != nil {
		return errors.Wrap(err, "Could not set defaults for processes monitor")
	}

	return am.SetConfigurationAndRun(conf)
}

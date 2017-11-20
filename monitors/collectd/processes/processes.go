package processes

//go:generate collectd-template-to-go processes.tmpl

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/processes"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewStaticMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig
	Processes            []string          `yaml:"processes"`
	ProcessMatch         map[string]string `yaml:"processMatch"`
	CollectContextSwitch bool              `yaml:"collectContextSwitch" default:"false"`
	ProcFSPath           string            `yaml:"procFSPath"`
}

// Validate will check the config for correctness.
func (c *Config) Validate() error {
	return nil
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.StaticMonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) bool {
	// ProcFSPath is a global config setting that gets propagated to each
	// monitor config, but allow overriding it if desired.
	if conf.ProcFSPath == "" {
		conf.ProcFSPath = conf.MonitorConfig.ProcFSPath
	}

	return am.SetConfigurationAndRun(conf)
}

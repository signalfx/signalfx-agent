package hadoop

import (
	"fmt"

	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"

	"github.com/signalfx/signalfx-agent/internal/core/config"

	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/hadoop"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			python.PyMonitor{
				MonitorCore: pyrunner.New("sfxcollectd"),
			},
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	pyConf               *python.Config
	// Resource Manager Hostname
	Host string `yaml:"host" validate:"required"`
	// Resource Manager Port
	Port uint16 `yaml:"port" validate:"required"`
	// Log verbose information about the plugin
	Verbose *bool `yaml:"verbose"`
}

// PythonConfig returns the embedded python.Config struct from the interface
func (c *Config) PythonConfig() *python.Config {
	return c.pyConf
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.PyMonitor
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	conf.pyConf = &python.Config{
		MonitorConfig: conf.MonitorConfig,
		Host:          conf.Host,
		Port:          conf.Port,
		ModuleName:    "hadoop_plugin",
		ModulePaths:   []string{collectd.MakePath("hadoop")},
		TypesDBPaths:  []string{collectd.MakePath("types.db")},
		PluginConfig: map[string]interface{}{
			"ResourceManagerURL":  fmt.Sprintf("http://%s", conf.Host),
			"ResourceManagerPort": conf.Port,
			"Interval":            conf.IntervalSeconds,
			"Verbose":             conf.Verbose,
		},
	}

	return m.PyMonitor.Configure(conf)
}

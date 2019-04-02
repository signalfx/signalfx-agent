package zookeeper

import (
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"

	"github.com/signalfx/signalfx-agent/internal/core/config"

	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/zookeeper"

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
	Host                 string `yaml:"host" validate:"required"`
	Port                 uint16 `yaml:"port" validate:"required"`
	Name                 string `yaml:"name"`
}

// PythonConfig returns the embedded python.Config struct from the interface
func (c *Config) PythonConfig() *python.Config {
	return c.pyConf
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.PyMonitor
}

// Configure configures and runs the plugin in python
func (rm *Monitor) Configure(conf *Config) error {
	conf.pyConf = &python.Config{
		MonitorConfig: conf.MonitorConfig,
		ModuleName:    "zk-collectd",
		ModulePaths:   []string{collectd.MakePath("zookeeper")},
		TypesDBPaths:  []string{collectd.MakePath("types.db")},
		Host:          conf.Host,
		Port:          conf.Port,
		PluginConfig: map[string]interface{}{
			"Hosts":    conf.Host,
			"Port":     conf.Port,
			"Instance": conf.Name,
		},
	}

	return rm.PyMonitor.Configure(conf)
}

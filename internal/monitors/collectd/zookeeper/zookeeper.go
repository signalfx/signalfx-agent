// +build !windows

package zookeeper

import (
	"os"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/zookeeper"

// MONITOR(collectd/zookeeper): Monitors an Apache Zookeeper instance.
//
// See the [Python plugin
// source](https://github.com/signalfx/collectd-zookeeper) and the
// [integrations repo
// page](https://github.com/signalfx/integrations/tree/master/collectd-zookeeper)
// for more information.

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			python.Monitor{
				MonitorCore: pyrunner.New("sfxcollectd"),
			},
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	// By not embedding python.Config we can override struct fields (i.e. Host and Port)
	// and add monitor specific config doc and struct tags.
	pyConfig *python.Config
	Host     string `yaml:"host" validate:"required"`
	Port     uint16 `yaml:"port" validate:"required"`
	Name     string `yaml:"name"`
}

// PythonConfig returns the python.Config struct contained in the config struct
func (c *Config) PythonConfig() *python.Config {
	return c.pyConfig
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.Monitor
}

// Configure configures and runs the plugin in python
func (rm *Monitor) Configure(conf *Config) error {
	conf.pyConfig = &python.Config{
		MonitorConfig: conf.MonitorConfig,
		Host:          conf.Host,
		Port:          conf.Port,
		ModuleName:    "zk-collectd",
		ModulePaths:   []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "zookeeper")},
		PluginConfig: map[string]interface{}{
			"Hosts": conf.Host,
			"Port":  conf.Port,
		},
		TypesDBPaths: []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "types.db")},
	}

	if conf.Name != "" {
		conf.pyConfig.PluginConfig["Instance"] = conf.Name
	}

	return rm.Monitor.Configure(conf)
}

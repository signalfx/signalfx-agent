// +build !windows

package zookeeper

import (
	"os"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
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
	python.CoreConfig `yaml:",inline" acceptsEndpoints:"true"`
	Host              string `yaml:"host" validate:"required"`
	Port              uint16 `yaml:"port" validate:"required"`
	Name              string `yaml:"name"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.Monitor
}

// Configure configures and runs the plugin in python
func (rm *Monitor) Configure(conf *Config) error {
	conf.PluginConfig = map[string]interface{}{
		"Hosts": conf.Host,
		"Port":  conf.Port,
	}
	if conf.ModuleName == "" {
		conf.ModuleName = "zk-collectd"
	}
	if len(conf.ModulePaths) == 0 {
		conf.ModulePaths = []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "zookeeper")}
	}
	if len(conf.TypesDBPaths) == 0 {
		conf.TypesDBPaths = []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "types.db")}
	}
	if conf.Name != "" {
		conf.PluginConfig["Instance"] = conf.Name
	}

	return rm.Monitor.Configure(conf)
}

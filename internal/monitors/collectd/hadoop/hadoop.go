// +build !windows

package hadoop

import (
	"os"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/core/config"

	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/hadoop"

// MONITOR(collectd/hadoop): Collects metrics about a Hadoop cluster using the
// [collectd Hadoop Python plugin](https://github.com/signalfx/collectd-hadoop).
// Also see
// https://github.com/signalfx/integrations/tree/master/collectd-hadoop.
//
// The `collectd/hadoop` monitor will collect metrics from the Resource Manager
// REST API for the following:
// - Cluster Metrics
// - Cluster Scheduler
// - Cluster Applications
// - Cluster Nodes
// - MapReduce Jobs
//
// Sample YAML configuration:
//
// ```yaml
// monitors:
// - type: collectd/hadoop
//   host: 127.0.0.1
//   port: 8088
// ```
//
// If a remote JMX port is exposed in the hadoop cluster, then
// you may also configure the [collectd/hadoopjmx](https://github.com/signalfx/signalfx-agent/tree/master/docs/monitors/collectd/hadoopjmx)
// monitor to collect additional metrics about the hadoop cluster.
//

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
	Verbose bool `yaml:"verbose"`
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
		ModulePaths:   []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "hadoop")},
		TypesDBPaths:  []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "types.db")},
		PluginConfig: map[string]interface{}{
			"ResourceManagerURL":  "http://{{.Host}}",
			"ResourceManagerPort": conf.Port,
			"Interval":            conf.IntervalSeconds,
			"Verbose":             conf.Verbose,
		},
	}

	return m.PyMonitor.Configure(conf)
}

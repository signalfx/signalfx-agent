// +build !windows

package haproxy

import (
	"os"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/haproxy"

// MONITOR(collectd/haproxy): Monitors an HAProxy instance.
//
// See https://github.com/signalfx/integrations/tree/master/collectd-haproxy.

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
	Host              string   `yaml:"host" validate:"required"`
	Port              uint16   `yaml:"port" validate:"required"`
	ProxiesToMonitor  []string `yaml:"proxiesToMonitor"`
	ExcludedMetrics   []string `yaml:"excludedMetrics"`
	EnhancedMetrics   bool     `yaml:"enhancedMetrics"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.Monitor
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	conf.PluginConfig = map[string]interface{}{
		"Socket":          "{{.Host}}:{{.Port}}",
		"Interval":        conf.IntervalSeconds,
		"EnhancedMetrics": conf.EnhancedMetrics,
	}
	if conf.ModuleName == "" {
		conf.ModuleName = "haproxy"
	}
	if len(conf.ModulePaths) == 0 {
		conf.ModulePaths = []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "haproxy")}
	}
	if len(conf.TypesDBPaths) == 0 {
		conf.TypesDBPaths = []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "types.db")}
	}
	if len(conf.ProxiesToMonitor) > 0 {
		conf.PluginConfig["ProxyMonitor"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.ProxiesToMonitor,
		}
	}
	if len(conf.ExcludedMetrics) > 0 {
		conf.PluginConfig["ExcludeMetric"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.ExcludedMetrics,
		}
	}
	return m.Monitor.Configure(conf)
}

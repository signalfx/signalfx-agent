// +build !windows

package haproxy

import (
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"

	"github.com/signalfx/signalfx-agent/internal/core/config"

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
	Host                 string   `yaml:"host" validate:"required"`
	Port                 uint16   `yaml:"port" validate:"required"`
	ProxiesToMonitor     []string `yaml:"proxiesToMonitor"`
	ExcludedMetrics      []string `yaml:"excludedMetrics"`
	EnhancedMetrics      bool     `yaml:"enhancedMetrics"`
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
		ModuleName:    "haproxy",
		ModulePaths:   []string{collectd.MakePath("haproxy")},
		TypesDBPaths:  []string{collectd.MakePath("types.db")},
		PluginConfig: map[string]interface{}{
			"Socket":          "{{.Host}}:{{.Port}}",
			"Interval":        conf.IntervalSeconds,
			"EnhancedMetrics": conf.EnhancedMetrics,
		},
	}

	if len(conf.ProxiesToMonitor) > 0 {
		conf.pyConf.PluginConfig["ProxyMonitor"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.ProxiesToMonitor,
		}
	}
	if len(conf.ExcludedMetrics) > 0 {
		conf.pyConf.PluginConfig["ExcludeMetric"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.ExcludedMetrics,
		}
	}
	return m.PyMonitor.Configure(conf)
}

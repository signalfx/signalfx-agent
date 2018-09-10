// +build !windows

package haproxy

import (
	"os"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
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

	ProxiesToMonitor []string `yaml:"proxiesToMonitor"`
	ExcludedMetrics  []string `yaml:"excludedMetrics"`
	EnhancedMetrics  bool     `yaml:"enhancedMetrics"`
}

// PythonConfig returns the python.Config struct contained in the config struct
func (c *Config) PythonConfig() *python.Config {
	return c.pyConfig
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.Monitor
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	pConf := map[string]interface{}{
		"Socket":          "{{.Host}}:{{.Port}}",
		"Interval":        conf.IntervalSeconds,
		"EnhancedMetrics": conf.EnhancedMetrics,
	}
	if len(conf.ProxiesToMonitor) > 0 {
		pConf["ProxyMonitor"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.ProxiesToMonitor,
		}
	}
	if len(conf.ExcludedMetrics) > 0 {
		pConf["ExcludeMetric"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.ExcludedMetrics,
		}
	}
	conf.pyConfig = &python.Config{
		MonitorConfig: conf.MonitorConfig,
		Host:          conf.Host,
		Port:          conf.Port,
		ModuleName:    "haproxy",
		ModulePaths:   []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "haproxy")},
		TypesDBPaths:  []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "types.db")},
		PluginConfig:  pConf,
	}
	return m.Monitor.Configure(conf)
}

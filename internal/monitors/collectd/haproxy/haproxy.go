package haproxy

import (
	"errors"
	"fmt"
	"strings"

	"github.com/signalfx/golib/v3/pointer"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"

	"github.com/signalfx/signalfx-agent/internal/core/config"

	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/subproc"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} {
		return &Monitor{
			python.PyMonitor{
				MonitorCore: subproc.New(),
			},
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	python.CommonConfig  `yaml:",inline"`
	pyConf               *python.Config
	Host                 string `yaml:"host" validate:"required"`
	Port                 uint16 `yaml:"port"`
	// A list of all the pxname(s) or svname(s) that you want to monitor (e.g. `["http-in", "server1", "backend"]`)
	ProxiesToMonitor []string `yaml:"proxiesToMonitor"`
	// Deprecated.  Please use `datapointsToExclude` on the monitor config
	// block instead.
	ExcludedMetrics []string `yaml:"excludedMetrics"`
	EnhancedMetrics *bool    `yaml:"enhancedMetrics"`
}

// GetExtraMetrics accounts for the EnhancedMetrics config
func (c *Config) GetExtraMetrics() []string {
	if c.EnhancedMetrics != nil && *c.EnhancedMetrics {
		// More robust to just allow everything through in this case.
		return []string{"*"}
	}
	return nil
}

var _ config.ExtraMetrics = &Config{}

// PythonConfig returns the embedded python.Config struct from the interface
func (c *Config) PythonConfig() *python.Config {
	c.pyConf.CommonConfig = c.CommonConfig
	return c.pyConf
}

// Validate check config if host is TCP or socket. TCP requires port.
func (c *Config) Validate() error {
	if !strings.HasPrefix(c.Host, "/") && c.Port == 0 {
		return errors.New("when using TCP for HAProxy connection, port must be specified")
	}
	return nil
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.PyMonitor
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	if m.Output.HasAnyExtraMetrics() {
		conf.EnhancedMetrics = pointer.Bool(true)
	}

	socket := conf.Host
	if conf.Port != 0 {
		socket += fmt.Sprintf(":%d", conf.Port)
	}
	conf.pyConf = &python.Config{
		MonitorConfig: conf.MonitorConfig,
		Host:          conf.Host,
		Port:          conf.Port,
		ModuleName:    "haproxy",
		ModulePaths:   []string{collectd.MakePythonPluginPath("haproxy")},
		TypesDBPaths:  []string{collectd.DefaultTypesDBPath()},
		PluginConfig: map[string]interface{}{
			"Socket":          socket,
			"Interval":        conf.IntervalSeconds,
			"EnhancedMetrics": conf.EnhancedMetrics,
			"ProxyMonitor": map[string]interface{}{
				"#flatten": true,
				"values":   conf.ProxiesToMonitor,
			},
			"ExcludeMetric": map[string]interface{}{
				"#flatten": true,
				"values":   conf.ExcludedMetrics,
			},
		},
	}

	return m.PyMonitor.Configure(conf)
}

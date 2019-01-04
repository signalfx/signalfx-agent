package solr

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/solr"

// MONITOR(collectd/solr): Monitors Solr instances.
//
// See https://github.com/signalfx/collectd-solr and
// https://github.com/signalfx/integrations/tree/master/collectd-solr

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
	// Cluster name of this solr cluster.
	Cluster string `yaml:"cluster"`
	// EnhancedMetrics boolean to indicate whether stats from /metrics are needed
	EnhancedMetrics *bool `yaml:"enhancedMetrics" default:"false"`
	// IncludeMetric metric name from the /admin/metrics endpoint to include (valid when EnhancedMetrics is "false")
	IncludeMetric string `yaml:"includeMetric"`
	// ExcludeMetric metric name from the /admin/metrics endpoint to exclude (valid when EnhancedMetrics is "true")
	ExcludeMetric string `yaml:"excludeMetric"`
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
		ModuleName:    "solr_collectd",
		ModulePaths:   []string{collectd.MakePath("solr")},
		TypesDBPaths:  []string{collectd.MakePath("types.db")},
		PluginConfig: map[string]interface{}{
			"Host":            conf.Host,
			"Port":            conf.Port,
			"Cluster":         conf.Cluster,
			"EnhancedMetrics": conf.EnhancedMetrics,
			"IncludeMetric":   conf.IncludeMetric,
			"ExcludeMetric":   conf.ExcludeMetric,
			"Interval":        conf.IntervalSeconds,
		},
	}

	return m.PyMonitor.Configure(conf)
}

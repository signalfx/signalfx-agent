// +build !windows

package elasticsearch

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"

	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/elasticsearch"

// MONITOR(collectd/elasticsearch): Monitors ElasticSearch instances.
//
// See https://github.com/signalfx/collectd-elasticsearch and
// https://github.com/signalfx/integrations/tree/master/collectd-elasticsearch

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
	// AdditionalMetrics to report on
	AdditionalMetrics []string `yaml:"additionalMetrics"`
	// DetailedMetrics turns on additional metric time series
	DetailedMetrics bool `yaml:"detailedMetrics" default:"true"`
	// EnableClusterHealth enables reporting on the cluster health
	EnableClusterHealth bool `yaml:"enableClusterHealth" default:"true"`
	// EnableIndexStats reports metrics about indexes
	EnableIndexStats bool `yaml:"enableIndexStats" default:"true"`
	// Indexes to report on
	Indexes []string `yaml:"indexes" default:"[\"_all\"]"`
	// IndexInterval is an interval in seconds at which the plugin will report index stats.
	// It must be greater than or equal, and divisible by the Interval configuration
	IndexInterval uint `yaml:"indexInterval" default:"300"`
	// IndexStatsMasterOnly sends index stats from the master only
	IndexStatsMasterOnly bool `yaml:"indexStatsMasterOnly" default:"false"`
	IndexSummaryOnly     bool `yaml:"indexSummaryOnly" default:"false"`
	// Password used to access elasticsearch stats api
	Password string `yaml:"password" neverLog:"true"`
	// Protocol used to connect: http or https
	Protocol string `yaml:"protocol"`
	// ThreadPools to report on
	ThreadPools []string `yaml:"threadPools" default:"[\"search\", \"index\"]"`
	// Username used to access elasticsearch stats api
	Username string `yaml:"username"`
	Version  string `yaml:"version"`
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
		ModuleName:    "elasticsearch_collectd",
		ModulePaths:   []string{collectd.MakePath("elasticsearch")},
		TypesDBPaths:  []string{collectd.MakePath("types.db")},
		PluginConfig: map[string]interface{}{
			"Host":                 conf.Host,
			"Port":                 conf.Port,
			"DetailedMetrics":      conf.DetailedMetrics,
			"EnableClusterHealth":  conf.EnableClusterHealth,
			"EnableIndexStats":     conf.EnableIndexStats,
			"IndexInterval":        conf.IndexInterval,
			"IndexStatsMasterOnly": conf.IndexStatsMasterOnly,
			"IndexSummaryOnly":     conf.IndexSummaryOnly,
			"Interval":             conf.IntervalSeconds,
			"Verbose":              false,
		},
	}

	if len(conf.AdditionalMetrics) > 0 {
		conf.pyConf.PluginConfig["AdditionalMetrics"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.AdditionalMetrics,
		}
	}
	if len(conf.Indexes) > 0 {
		conf.pyConf.PluginConfig["Indexes"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.Indexes,
		}
	}
	if conf.Password != "" {
		conf.pyConf.PluginConfig["Password"] = conf.Password
	}
	if conf.Protocol != "" {
		conf.pyConf.PluginConfig["Protocol"] = conf.Protocol
	}
	if conf.Username != "" {
		conf.pyConf.PluginConfig["Username"] = conf.Username
	}
	if len(conf.ThreadPools) > 0 {
		conf.pyConf.PluginConfig["ThreadPools"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.ThreadPools,
		}
	}
	if conf.Version != "" {
		conf.pyConf.PluginConfig["Version"] = conf.Version
	}

	return m.PyMonitor.Configure(conf)
}

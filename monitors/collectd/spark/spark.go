package spark

//go:generate collectd-template-to-go spark.tmpl

import (
	"errors"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/spark"

type sparkClusterType string

const (
	sparkStandalone sparkClusterType = "Standalone"
	sparkMesos                       = "Mesos"
)

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"true"`

	Host                      string           `yaml:"host"`
	Port                      uint16           `yaml:"port"`
	Name                      string           `yaml:"name"`
	IsMaster                  bool             `yaml:"isMaster" default:"false"`
	ClusterType               sparkClusterType `yaml:"clusterType"`
	CollectApplicationMetrics bool             `yaml:"collectApplicationMetrics" default:"false"`
	EnhancedMetrics           bool             `yaml:"enhancedMetrics" default:"false"`
	MetricsToInclude          []string         `yaml:"metricsToInclude" default:"[]"`
	MetricsToExclude          []string         `yaml:"metricsToExclude" default:"[]"`
}

// Validate will check the config for correctness.
func (c *Config) Validate() error {
	if c.CollectApplicationMetrics && !c.IsMaster {
		return errors.New("Cannot collect application metrics from non-master endpoint")
	}

	if c.ClusterType == "" {
		return errors.New("clusterType is required for Spark monitors")
	}
	return nil
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) error {
	return am.SetConfigurationAndRun(conf)
}

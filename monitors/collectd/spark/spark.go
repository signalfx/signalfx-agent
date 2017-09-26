package spark

//go:generate collectd-template-to-go spark.tmpl

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
	log "github.com/sirupsen/logrus"
)

const monitorType = "collectd/spark"

type sparkClusterType string

const (
	sparkStandalone sparkClusterType = "standalone"
	sparkMesos                       = "mesos"
)

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewServiceMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig
	IsMaster                  bool                    `yaml:"isMaster" default:"false"`
	ClusterType               sparkClusterType        `yaml:"clusterType" required:"true"`
	CollectApplicationMetrics bool                    `yaml:"collectApplicationMetrics" default:"false"`
	EnhancedMetrics           bool                    `yaml:"enhancedMetrics" default:"false"`
	MetricsToInclude          []string                `yaml:"metricsToInclude" default:"[]"`
	MetricsToExclude          []string                `yaml:"metricsToExclude" default:"[]"`
	ServiceEndpoints          []services.EndpointCore `yaml:"serviceEndpoints" default:"[]"`
}

func (c *Config) Validate() bool {
	if c.CollectApplicationMetrics && !c.IsMaster {
		log.WithFields(log.Fields{
			"monitorType": monitorType,
		}).Error("Cannot collect application metrics from non-master endpoint")
		return false
	}
	return true
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.ServiceMonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) bool {
	return am.SetConfigurationAndRun(conf)
}

package elasticsearch

//go:generate collectd-template-to-go elasticsearch.tmpl

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/elasticsearch"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewServiceMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

type serviceEndpoint struct {
	services.EndpointCore `yaml:",inline"`
	// AdditionalMetrics to report on
	AdditionalMetrics []string `yaml:"additionalMetrics"`
	// DetailedMetrics turns on additional metric time series
	DetailedMetrics *bool `yaml:"detailedMetrics"`
	// EnableClusterHealth enables reporting on the cluster health
	EnableClusterHealth *bool `yaml:"enableClusterHealth"`
	// EnableIndexStats reports metrics about indexes
	EnableIndexStats *bool `yaml:"enableIndexStats"`
	// Indexes to report on
	Indexes []string `yaml:"indexes"`
	// IndexInterval is an interval in seconds at which the plugin will report index stats.
	// It must be greater than or equal, and divisible by the Interval configuration
	IndexInterval *uint `yaml:"indexInterval"`
	// IndexStatsMasterOnly sends index stats from the master only
	IndexStatsMasterOnly *bool `yaml:"indexStatsMasterOnly"`
	IndexSummaryOnly     *bool `yaml:"indexSummaryOnly"`
	// Password used to access elasticsearch stats api
	Password *string `yaml:"password"`
	// Protocol used to connect: http or https
	Protocol *string `yaml:"protocol"`
	// ThreadPools to report on
	ThreadPools []string `yaml:"threadPools"`
	// Username used to access elasticsearch stats api
	Username *string `yaml:"username"`
	Version  *string `yaml:"version"`
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig
	CommonEndpointConfig serviceEndpoint   `yaml:",inline" default:"{}"`
	ServiceEndpoints     []serviceEndpoint `yaml:"serviceEndpoints" default:"[]"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.ServiceMonitorCore
}

// Configure configures and runs the plugin in collectd
func (em *Monitor) Configure(conf *Config) bool {
	return em.SetConfigurationAndRun(&conf.MonitorConfig, &conf.CommonEndpointConfig)
}

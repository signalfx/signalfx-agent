package elasticsearch

//go:generate collectd-template-to-go elasticsearch.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/elasticsearch"

// MONITOR(collectd/elasticsearch): Monitors ElasticSearch instances.
//
// See https://github.com/signalfx/collectd-elasticsearch and
// https://github.com/signalfx/integrations/tree/master/collectd-elasticsearch

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

	Host string `yaml:"host" validate:"required"`
	Port uint16 `yaml:"port" validate:"required"`
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
	Password string `yaml:"password"`
	// Protocol used to connect: http or https
	Protocol string `yaml:"protocol"`
	// ThreadPools to report on
	ThreadPools []string `yaml:"threadPools" default:"[\"search\", \"index\"]"`
	// Username used to access elasticsearch stats api
	Username string `yaml:"username"`
	Version  string `yaml:"version"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	return m.SetConfigurationAndRun(conf)
}

package etcd

//go:generate collectd-template-to-go etcd.tmpl

import (
	"errors"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/etcd"

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
	ClusterName       string                  `yaml:"clusterName" required:"true"`
	SSLKeyFile        *string                 `yaml:"sslKeyFile"`
	SSLCertificate    *string                 `yaml:"sslCertificate"`
	SSLCACerts        *string                 `yaml:"sslCACerts"`
	SSLCertValidation bool                    `yaml:"sslCertValidation" default:"true"`
	EnhancedMetrics   bool                    `yaml:"enhancedMetrics" default:"false"`
	MetricsToInclude  []string                `yaml:"metricsToInclude" default:"[]"`
	MetricsToExclude  []string                `yaml:"metricsToExclude" default:"[]"`
	ServiceEndpoints  []services.EndpointCore `yaml:"serviceEndpoints" default:"[]"`
}

// Validate will check the config before the monitor is instantiated
func (c *Config) Validate() error {
	if c.ClusterName == "" {
		return errors.New("Etcd monitor config requires a clusterName")
	}
	return nil
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.ServiceMonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) bool {
	return am.SetConfigurationAndRun(conf)
}

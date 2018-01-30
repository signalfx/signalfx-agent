package etcd

//go:generate collectd-template-to-go etcd.tmpl

import (
	"github.com/creasty/defaults"
	"github.com/pkg/errors"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/etcd"

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

	Host              string   `yaml:"host"`
	Port              uint16   `yaml:"port"`
	Name              string   `yaml:"name"`
	ClusterName       string   `yaml:"clusterName" required:"true"`
	SSLKeyFile        *string  `yaml:"sslKeyFile"`
	SSLCertificate    *string  `yaml:"sslCertificate"`
	SSLCACerts        *string  `yaml:"sslCACerts"`
	SSLCertValidation *bool    `yaml:"sslCertValidation" default:"true"`
	EnhancedMetrics   bool     `yaml:"enhancedMetrics"`
	MetricsToInclude  []string `yaml:"metricsToInclude"`
	MetricsToExclude  []string `yaml:"metricsToExclude"`
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
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) error {
	if err := defaults.Set(conf); err != nil {
		return errors.Wrap(err, "Could not set defaults for etcd monitor")
	}
	return am.SetConfigurationAndRun(conf)
}

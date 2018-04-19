package etcd

//go:generate collectd-template-to-go etcd.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/etcd"

// MONITOR(collectd/etcd): Monitors an etcd key/value store.
//
// See https://github.com/signalfx/integrations/tree/master/collectd-etcd and
// https://github.com/signalfx/collectd-etcd

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
	// An arbitrary name of the etcd cluster to make it easier to group
	// together and identify instances.
	ClusterName       string `yaml:"clusterName" validate:"required"`
	SSLKeyFile        string `yaml:"sslKeyFile"`
	SSLCertificate    string `yaml:"sslCertificate"`
	SSLCACerts        string `yaml:"sslCACerts"`
	SkipSSLValidation bool   `yaml:"skipSSLValidation"`
	EnhancedMetrics   bool   `yaml:"enhancedMetrics"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) error {
	return am.SetConfigurationAndRun(conf)
}

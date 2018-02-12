package consul

//go:generate collectd-template-to-go consul.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/consul"

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
	Name string `yaml:"name"`

	ACLToken            string `yaml:"aclToken"`
	UseHTTPS            bool   `yaml:"useHTTPS"`
	EnhancedMetrics     bool   `yaml:"enhancedMetrics"`
	CACertificate       string `yaml:"caCertificate"`
	ClientCertificate   string `yaml:"clientCertificate"`
	ClientKey           string `yaml:"clientKey"`
	SignalFxAccessToken string `yaml:"signalFxAccessToken"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) error {
	return am.SetConfigurationAndRun(conf)
}

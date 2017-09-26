package rabbitmq

//go:generate collectd-template-to-go rabbitmq.tmpl

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/rabbitmq"

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
	CollectChannels  *bool                   `yaml:"collectChannels"`
	CollectExchanges *bool                   `yaml:"collectExchanges"`
	CollectNodes     *bool                   `yaml:"collectNodes"`
	CollectQueues    *bool                   `yaml:"collectQueues"`
	HTTPTimeout      *int                    `yaml:"httpTimeout"`
	VerbosityLevel   *string                 `yaml:"verbosityLevel"`
	Username         *string                 `yaml:"username"`
	Password         *string                 `yaml:"password"`
	ServiceEndpoints []services.EndpointCore `yaml:"serviceEndpoints" default:"[]"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.ServiceMonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) bool {
	return am.SetConfigurationAndRun(conf)
}

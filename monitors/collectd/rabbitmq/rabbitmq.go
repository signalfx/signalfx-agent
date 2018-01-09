package rabbitmq

//go:generate collectd-template-to-go rabbitmq.tmpl

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/rabbitmq"

func init() {
	monitors.Register(monitorType, func(id monitors.MonitorID) interface{} {
		return &Monitor{
			*collectd.NewMonitorCore(id, CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"true"`

	Host             string  `yaml:"host"`
	Port             uint16  `yaml:"port"`
	Name             string  `yaml:"name"`

	CollectChannels    *bool                   `yaml:"collectChannels"`
	CollectConnections *bool                   `yaml:"collectConnections"`
	CollectExchanges   *bool                   `yaml:"collectExchanges"`
	CollectNodes       *bool                   `yaml:"collectNodes"`
	CollectQueues      *bool                   `yaml:"collectQueues"`
	HTTPTimeout        *int                    `yaml:"httpTimeout"`
	VerbosityLevel     *string                 `yaml:"verbosityLevel"`
	Username           *string                 `yaml:"username"`
	Password           *string                 `yaml:"password"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) error {
	return am.SetConfigurationAndRun(conf)
}

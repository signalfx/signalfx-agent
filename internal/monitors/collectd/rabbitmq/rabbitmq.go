package rabbitmq

//go:generate collectd-template-to-go rabbitmq.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/rabbitmq"

// MONITOR(collectd/rabbitmq): Monitors an instance of RabbitMQ using the
// [collectd RabbitMQ Python
// Plugin](https://github.com/signalfx/collectd-rabbitmq).
//
// See the [integration
// doc](https://github.com/signalfx/integrations/tree/master/collectd-rabbitmq)
// for more information.

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

	CollectChannels    bool   `yaml:"collectChannels"`
	CollectConnections bool   `yaml:"collectConnections"`
	CollectExchanges   bool   `yaml:"collectExchanges"`
	CollectNodes       bool   `yaml:"collectNodes"`
	CollectQueues      bool   `yaml:"collectQueues"`
	HTTPTimeout        int    `yaml:"httpTimeout"`
	VerbosityLevel     string `yaml:"verbosityLevel"`
	Username           string `yaml:"username" validate:"required"`
	Password           string `yaml:"password" validate:"required" neverLog:"true"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) error {
	return am.SetConfigurationAndRun(conf)
}

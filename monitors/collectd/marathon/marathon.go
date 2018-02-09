package marathon

//go:generate collectd-template-to-go marathon.tmpl

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/marathon"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	// Make this single instance since we can't add dimensions
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true" singleInstance:"true"`

	Host     string `yaml:"host" validate:"required"`
	Port     uint16 `yaml:"port" validate:"required"`
	Name     string `yaml:"name"`
	Username string `yaml:"username"`
	Password string `yaml:"password" neverLog:"true"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) error {
	return am.SetConfigurationAndRun(conf)
}

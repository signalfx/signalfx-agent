package haproxy

//go:generate collectd-template-to-go haproxy.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/haproxy"

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

	ProxiesToMonitor []string `yaml:"proxiesToMonitor"`
	ExcludedMetrics  []string `yaml:"excludedMetrics"`
	EnhancedMetrics  bool     `yaml:"enhancedMetrics"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (mm *Monitor) Configure(conf *Config) error {
	return mm.SetConfigurationAndRun(conf)
}

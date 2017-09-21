package redis

//go:generate collectd-template-to-go redis.tmpl

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/redis"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewServiceMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

type serviceEndpoint struct {
	services.EndpointCore `yaml:",inline"`
	Auth                  *string `yaml:"auth"`
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig
	CommonEndpointConfig serviceEndpoint   `yaml:",inline" default:"{}"`
	ServiceEndpoints     []serviceEndpoint `yaml:"serviceEndpoints" default:"[]"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.ServiceMonitorCore
}

// Configure configures and runs the plugin in collectd
func (rm *Monitor) Configure(conf *Config) bool {
	return rm.SetConfigurationAndRun(&conf.MonitorConfig, &conf.CommonEndpointConfig)
}

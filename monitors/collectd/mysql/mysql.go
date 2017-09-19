package mysql

//go:generate collectd-template-to-go mysql.tmpl

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/mysql"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewServiceMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

type serviceEndpoint struct {
	services.EndpointCore `yaml:",inline"`
	Databases             []struct {
		Name     string  `yaml:"name"`
		Username string  `yaml:"username"`
		Password *string `yaml:"password"`
	} `yaml:"databases" required:"true"`
	// These credentials serve as defaults for all databases if not overridden
	Username string  `yaml:"username"`
	Password *string `yaml:"password"`

	ReportHost *bool `yaml:"reportHost"`
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
func (am *Monitor) Configure(conf *Config) bool {
	return am.SetConfigurationAndRun(&conf.MonitorConfig, &conf.CommonEndpointConfig)
}

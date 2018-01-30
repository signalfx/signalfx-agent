package nginx

//go:generate collectd-template-to-go nginx.tmpl

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/nginx"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"true"`

	Host     string  `yaml:"host" validate:"required"`
	Port     uint16  `yaml:"port" validate:"required"`
	Name     string  `yaml:"name"`
	URL      string  `yaml:"url" default:"http://{{.Host}}:{{.Port}}/nginx_status" help:"The full URL of the status endpoint; can be a template"`
	Username *string `yaml:"username"`
	Password *string `yaml:"password"`
	Timeout  *int    `yaml:"timeout"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	return m.SetConfigurationAndRun(conf)
}

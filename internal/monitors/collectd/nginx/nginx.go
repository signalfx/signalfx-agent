// +build !windows

package nginx

//go:generate collectd-template-to-go nginx.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/nginx"

// MONITOR(collectd/nginx): Monitors an nginx instance using our fork of the
// collectd nginx plugin based on the [collectd nginx
// plugin](https://collectd.org/wiki/index.php/Plugin:nginx).
//
// See the [integrations
// doc](https://github.com/signalfx/integrations/tree/master/collectd-nginx)
// for more information.

// CUMULATIVE(connections.accepted): Connections accepted by Nginx Web Server
// CUMULATIVE(connections.handled): Connections handled by Nginx Web Server
// GAUGE(nginx_connections.active): Connections active in Nginx Web Server
// GAUGE(nginx_connections.reading): Connections being read by Nginx Web Server
// GAUGE(nginx_connections.waiting): Connections waited on by Nginx Web Server
// GAUGE(nginx_connections.writing): Connections being written by Nginx Web Server
// CUMULATIVE(nginx_requests): Requests handled by Nginx Web Server

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
	// The full URL of the status endpoint; can be a template
	URL      string `yaml:"url" default:"http://{{.Host}}:{{.Port}}/nginx_status"`
	Username string `yaml:"username"`
	Password string `yaml:"password" neverLog:"true"`
	Timeout  int    `yaml:"timeout"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	return m.SetConfigurationAndRun(conf)
}

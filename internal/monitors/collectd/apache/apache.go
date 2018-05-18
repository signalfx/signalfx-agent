// +build !windows

package apache

//go:generate collectd-template-to-go apache.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/apache"

// MONITOR(collectd/apache): Monitors Apache webservice instances using
// the information provided by `mod_status`.
//
// See https://github.com/signalfx/integrations/tree/master/collectd-apache
//
// Sample YAML configuration:
//
//
// ```
// monitors:
//  - type: collectd/apache
//    host: localhost
//    port: 80
// ```
//
// If `mod_status` is exposed on an endpoint other than `/mod_status`, you can
// use the `url` config option to specify the path:
//
// ```
// monitors:
//  - type: collectd/apache
//    host: localhost
//    port: 80
//    url: "http://{{.Host}}:{{.Port}}/server-status?auto"
// ```

// CUMULATIVE(apache_bytes): Bytes served by Apache
// GAUGE(apache_connections): Connections served by Apache
// GAUGE(apache_idle_workers): Apache workers that are idle
// CUMULATIVE(apache_requests): Requests served by Apache
// GAUGE(apache_scoreboard.closing): Number of workers in the process of
// closing connections
// GAUGE(apache_scoreboard.dnslookup): Number of workers performing DNS lookup
// GAUGE(apache_scoreboard.finishing): Number of workers that are finishing
// GAUGE(apache_scoreboard.idle_cleanup): Number of idle threads ready for cleanup
// GAUGE(apache_scoreboard.keepalive): Number of keep-alive connections
// GAUGE(apache_scoreboard.logging): Number of workers writing to log file
// GAUGE(apache_scoreboard.open): Number of worker thread slots that are open
// GAUGE(apache_scoreboard.reading): Number of workers reading requests
// GAUGE(apache_scoreboard.sending): Number of workers sending responses
// GAUGE(apache_scoreboard.starting): Number of workers starting up
// GAUGE(apache_scoreboard.waiting): Number of workers waiting for requests

// DIMENSION(plugin_instance): Set to whatever you set in the `name` config option.

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

	// The hostname of the Apache server
	Host string `yaml:"host" validate:"required"`
	// The port number of the Apache server
	Port uint16 `yaml:"port" validate:"required"`
	// This will be sent as the `plugin_instance` dimension and can be any name
	// you like.
	Name string `yaml:"name"`
	// You can specify a username and password to do basic HTTP auth

	// The URL, either a final url or a Go template that will be populated with
	// the host and port values.
	URL      string `yaml:"url" default:"http://{{.Host}}:{{.Port}}/mod_status?auto"`
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

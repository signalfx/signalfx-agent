// +build linux

package php

//go:generate ../../../../scripts/collectd-template-to-go php.tmpl

import (
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/collectd"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} {
		return &Monitor{
			*collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	// The hostname of the webserver (i.e. `127.0.0.1`)
	Host string `yaml:"host" validate:"required"`
	// The port number of the webserver (i.e. `80`)
	Port uint16 `yaml:"port" validate:"required"`
	// This will be sent as the `plugin_instance` dimension and can be any name
	// you like.
	Name string `yaml:"name"`
	// The URL, either a final URL or a Go template that will be populated with
	// the `host` and `port` values.
	URL string `yaml:"url" default:"http://{{.Host}}:{{.Port}}/status?json"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) error {
	return am.SetConfigurationAndRun(conf)
}

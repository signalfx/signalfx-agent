package docker

//go:generate collectd-template-to-go docker.tmpl

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/docker"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewServiceMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig
	DockerURL      *string  `yaml:"dockerURL"`
	ExcludedImages []string `yaml:"excludedImages"`
	ExcludedNames  []string `yaml:"excludedNames"`
	ExcludedLabels []string `yaml:"excludedLabels"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.ServiceMonitorCore
}

// Configure configures and runs the plugin in collectd
func (rm *Monitor) Configure(conf *Config) bool {
	return rm.SetConfigurationAndRun(conf)
}

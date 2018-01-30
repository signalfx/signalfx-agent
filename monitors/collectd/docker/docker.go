package docker

//go:generate collectd-template-to-go docker.tmpl

import (
	"errors"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
)

const monitorType = "collectd/docker"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig
	DockerURL      string            `yaml:"dockerURL"`
	ExcludedImages []string          `yaml:"excludedImages"`
	ExcludedNames  []string          `yaml:"excludedNames"`
	ExcludedLabels map[string]string `yaml:"excludedLabels"`
}

// Validate will check the config before the monitor is instantiated
func (c *Config) Validate() error {
	if len(c.DockerURL) == 0 {
		return errors.New("dockerURL must be specified in the docker monitor")
	}
	return nil
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (rm *Monitor) Configure(conf *Config) error {
	return rm.SetConfigurationAndRun(conf)
}

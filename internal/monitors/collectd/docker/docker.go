package docker

//go:generate collectd-template-to-go docker.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/docker"

// MONITOR(collectd/docker): Pulls container stats from the Docker Engine.
//
// See https://github.com/signalfx/docker-collectd-plugin.

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
	// A set of dimensions to add to container metrics (see
	// https://github.com/signalfx/docker-collectd-plugin#extracting-additional-dimensions).
	Dimensions map[string]string `yaml:"dimensions"`
	// URL of the Docker engine, can be a unix socket path.
	DockerURL           string            `yaml:"dockerURL" validate:"required"`
	ExcludedImages      []string          `yaml:"excludedImages"`
	ExcludedNames       []string          `yaml:"excludedNames"`
	ExcludedLabels      map[string]string `yaml:"excludedLabels"`
	CollectNetworkStats bool              `yaml:"collectNetworkStats"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (rm *Monitor) Configure(conf *Config) error {
	return rm.SetConfigurationAndRun(conf)
}

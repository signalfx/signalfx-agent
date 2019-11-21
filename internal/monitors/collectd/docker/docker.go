// +build linux

package docker

//go:generate ../../../../scripts/collectd-template-to-go docker.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
	log "github.com/sirupsen/logrus"
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
	config.MonitorConfig
	// A set of dimensions to add to container metrics (see
	// https://github.com/signalfx/docker-collectd-plugin#extracting-additional-dimensions).
	Dimensions map[string]string `yaml:"dimensions"`
	// URL of the Docker engine, can be a unix socket path.
	DockerURL string `yaml:"dockerURL" validate:"required"`
	// A list of images to exclude from monitoring
	ExcludedImages []string `yaml:"excludedImages"`
	// A list of container names to exclude from monitoring
	ExcludedNames []string `yaml:"excludedNames"`
	// A map of label keys/values that will cause a container to be ignored.
	ExcludedLabels map[string]string `yaml:"excludedLabels"`
	// If true, will collect network stats about a container (will not work in
	// some environments like Kubernetes).
	CollectNetworkStats bool `yaml:"collectNetworkStats"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (rm *Monitor) Configure(conf *Config) error {
	log.Warn("The collectd/docker monitor is deprecated in favor of the docker-container-stats monitor.")
	return rm.SetConfigurationAndRun(conf)
}

// GetExtraMetrics returns additional metrics that should be allowed through.
func (c *Config) GetExtraMetrics() []string {
	var extraMetrics []string

	if c.CollectNetworkStats {
		extraMetrics = append(extraMetrics, groupMetricsMap[groupNetwork]...)
	}
	return extraMetrics
}

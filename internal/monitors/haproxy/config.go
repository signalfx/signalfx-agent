package haproxy

import (
	"time"

	"github.com/signalfx/signalfx-agent/internal/core/config"
)

// Config is the config for this monitor.
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	// Basic Auth username to use on each request, if any.
	Username string `yaml:"username"`
	// Basic Auth password to use on each request, if any.
	Password string `yaml:"password" neverLog:"true"`
	// URI on which to scrape HAProxy.
	ScrapeURI string `yaml:"scrapeURI" default:"http://localhost/;csv"`
	// Flag that enables SSL certificate verification for the scrape URI.
	SSLVerify bool `yaml:"sslVerify" default:"true"`
	// Timeout for trying to get stats from HAProxy. This should be a duration string that is accepted by https://golang.org/pkg/time/#ParseDuration
	Timeout time.Duration `yaml:"timeout" default:"5s"`
	// Flag that enables renaming haproxy stats to equivalent SignalFx metric names.
	UseSignalFxMetricNames bool `yaml:"useSignalFxMetricNames" default:"true"`
}

// GetMonitorType is a ConfigInterface method implementation for getting the monitor type.
func (c *Config) GetMonitorType() string {
	return monitorType
}

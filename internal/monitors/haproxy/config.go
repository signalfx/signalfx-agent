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
	// URL on which to scrape HAProxy. The URL scheme `http://` is designated as http-type configuration while `unix://` socket-type.
	URL string `yaml:"url" validate:"required"`
	// Flag that enables SSL certificate verification for the scrape URI.
	SSLVerify bool `yaml:"sslVerify" default:"true"`
	// Timeout for trying to get stats from HAProxy. This should be a duration string that is accepted by https://golang.org/pkg/time/#ParseDuration
	Timeout time.Duration `yaml:"timeout" default:"5s"`
	// A list of the pxname(s) and svname(s) to monitor (e.g. `["http-in", "server1", "backend"]`).
	// If empty then metrics of all proxies will be reported.
	Proxies []string `yaml:"proxies"`
}

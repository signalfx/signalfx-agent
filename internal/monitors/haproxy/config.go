package haproxy

import (
	"fmt"
	"time"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	logger "github.com/sirupsen/logrus"
)

// Config is the config for this monitor.
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

	// The host/ip address of the HAProxy instance. This is used to construct
	// the `url` option if not provided.
	Host string `yaml:"host"`
	// The port of the HAProxy instance's stats endpoint (if using HTTP). This
	// is used to construct the `url` option if not provided.
	Port uint16 `yaml:"port"`
	// Whether to connect on HTTPS or HTTP. If you want to use a UNIX socket,
	// then specify the `url` config option with the format `unix://...` and
	// omit `host`, `port` and `useHTTPS`.
	UseHTTPS bool `yaml:"useHTTPS"`
	// The path to HAProxy stats. The default is `stats?stats;csv`.
	// This is used to construct the `url` option if not provided.
	Path string `yaml:"path" default:"stats?stats;csv"`

	// URL on which to scrape HAProxy. Scheme `http://` for http-type and
	// `unix://` socket-type urls.  If this is not provided, it will be derive
	// from the `host`, `port`, and `useHTTPS` options.
	URL string `yaml:"url"`
	// Basic Auth username to use on each request, if any.
	Username string `yaml:"username"`
	// Basic Auth password to use on each request, if any.
	Password string `yaml:"password" neverLog:"true"`
	// Flag that enables SSL certificate verification for the scrape URL.
	SSLVerify bool `yaml:"sslVerify" default:"true"`
	// Timeout for trying to get stats from HAProxy. This should be a duration string that is accepted by https://golang.org/pkg/time/#ParseDuration
	Timeout time.Duration `yaml:"timeout" default:"5s"`
	// A list of the pxname(s) and svname(s) to monitor (e.g. `["http-in", "server1", "backend"]`).
	// If empty then metrics for all proxies will be reported.
	Proxies []string `yaml:"proxies"`
}

// Validate the config
func (c *Config) Validate() error {
	switch {
	case c.URL == "" && c.DiscoveryRule == "":
		return fmt.Errorf("must provide config value for url or discovery rule")
	case c.DiscoveryRule != "" && c.Path == "":
		return fmt.Errorf("must provide config value for path along with discovery rul")
	case c.URL != "" && c.DiscoveryRule != "":
		logger.Warnf("Discovery rule ignored. Do not provide config values for both url and discovery rule.")
	}
	return nil
}

func (c *Config) ScrapeURL() string {
	if c.URL != "" {
		return c.URL
	}
	scheme := "http"
	if c.UseHTTPS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d/%s", scheme, c.Host, c.Port, c.Path)
}

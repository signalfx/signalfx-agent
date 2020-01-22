package haproxy

import (
	"fmt"

	"github.com/signalfx/signalfx-agent/pkg/core/common/httpclient"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/utils/timeutil"
)

// Config is the config for this monitor.
type Config struct {
	config.MonitorConfig  `yaml:",inline" acceptsEndpoints:"true"`
	httpclient.HTTPConfig `yaml:",inline"`

	// The host/ip address of the HAProxy instance. This is used to construct the `url` option if not provided.
	Host string `yaml:"host"`
	// The port of the HAProxy instance's stats endpoint (if using HTTP). This is used to construct the `url` option if not provided.
	Port uint16 `yaml:"port"`
	// The path to HAProxy stats. The default is `stats?stats;csv`. This is used to construct the `url` option if not provided.
	Path string `yaml:"path" default:"stats?stats;csv"`
	// Whether to connect on HTTPS or HTTP. If you want to use a UNIX socket, then specify the `url` config option with the format `unix://...` and omit `host`, `port` and `useHTTPS`.
	URL string `yaml:"url"`
	// A list of the pxname(s) and svname(s) to monitor (e.g. `["http-in", "server1", "backend"]`). If empty then metrics for all proxies will be reported.
	Proxies []string `yaml:"proxies"`
	// Timeout when communicating over Unix sockets
	UnixTimeout timeutil.Duration `yaml:"unixTimeout" default:"10s"`
}

func (c *Config) ScrapeURL() string {
	if c.URL != "" {
		return c.URL
	}
	return fmt.Sprintf("%s://%s:%d/%s", c.Scheme(), c.Host, c.Port, c.Path)
}

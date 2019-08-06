package haproxy

import (
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/prometheusexporter"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config is the config for this monitor.
// Config implements ConfigInterface through prometheusexporter.Config.
type Config struct {
	prometheusexporter.Config                 `yaml:",inline" acceptsEndpoints:"true"`
	exporter                  *ExporterConfig `yaml:"exporter"`
}

// Validate k8s-specific configuration.
func (c *Config) Validate() error {
	return nil
}

// Monitor for prometheus exporter metrics
type Monitor struct {
	prometheusexporter.Monitor
}

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf Config) error {
	if conf.exporter != nil {
		conf.exporter.Run()
	}
	return m.Monitor.Configure(&conf)
}

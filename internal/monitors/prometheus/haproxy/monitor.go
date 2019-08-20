package haproxy

import (
	"context"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	exporter "github.com/signalfx/signalfx-agent/internal/monitors/prometheus/haproxy/prometheus"
	"github.com/signalfx/signalfx-agent/internal/monitors/prometheusexporter"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Monitor for Prometheus server metrics Exporter
type Monitor struct {
	prometheusexporter.Monitor
}

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	conf.SetExporterDefaults()
	m.Ctx, m.Cancel = context.WithCancel(context.Background())
	if conf.ExporterConfig != nil {
		exporter.StartServer(conf.ExporterConfig, m.Ctx)
	}
	return m.Monitor.Configure(conf)
}


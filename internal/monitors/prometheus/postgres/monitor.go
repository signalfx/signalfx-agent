package postgres

import (
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/prometheusexporter"
	log "github.com/sirupsen/logrus"
)

const monitorType = "prometheus/postgres"

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	prometheusexporter.Config `yaml:",inline"`
}

// Monitor for Prometheus Redis Exporter
type Monitor struct {
	prometheusexporter.Monitor
}

// Configure the underlying Prometheus exporter monitor
func (m *Monitor) Configure(conf *Config) error {
	return m.Monitor.Configure(&conf.Config)
}

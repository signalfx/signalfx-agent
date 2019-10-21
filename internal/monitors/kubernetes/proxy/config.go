package proxy

import "github.com/signalfx/signalfx-agent/internal/monitors/prometheusexporter"

func init() {
	prometheusexporter.RegisterMonitor(monitorMetadata)
}

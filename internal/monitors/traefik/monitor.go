package traefik

import (
	"github.com/signalfx/signalfx-agent/internal/monitors"
	pe "github.com/signalfx/signalfx-agent/internal/monitors/prometheusexporter"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &pe.Monitor{} }, &pe.Config{})
}

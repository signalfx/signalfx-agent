package traefik

import (
	pe "github.com/signalfx/signalfx-agent/internal/monitors/prometheusexporter"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &pe.Monitor{} }, &pe.Config{})
}

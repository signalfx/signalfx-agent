package metadata

import (
	"time"

	"github.com/signalfx/golib/event"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
)

// MetadataMonitor is a base monitor struct for emitting metadata properties
type MetadataMonitor struct {
	Output types.Output
}

// EmitProperty emits a property formatted as a signalfx metadata event
func (m *MetadataMonitor) EmitProperty(name string, val string) {
	m.Output.SendEvent(
		event.NewWithProperties(
			"objects.host-meta-data",
			event.AGENT,
			map[string]string{
				"plugin":   "signalfx-metadata",
				"severity": "4",
			},
			map[string]interface{}{
				"property": name,
				"message":  val,
			},
			time.Now()),
	)
}

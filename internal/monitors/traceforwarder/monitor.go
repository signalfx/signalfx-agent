package traceforwarder

import (
	"github.com/signalfx/signalfx-agent/internal/monitors/forwarder"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &forwarder.Monitor{} }, &forwarder.Config{})
}

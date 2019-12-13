package traceforwarder

import (
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/forwarder"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &forwarder.Monitor{} }, &forwarder.Config{})
}

package protocol

import (
	"io"

	"context"
	"github.com/signalfx/golib/datapoint/dpsink"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/golib/trace"
)

// DatapointForwarder can send datapoints and not events
type DatapointForwarder interface {
	sfxclient.Collector
	io.Closer
	dpsink.DSink
}

// Forwarder is the basic interface endpoints must support for the gateway to forward to them
type Forwarder interface {
	dpsink.Sink
	trace.Sink
	Pipeline
	sfxclient.Collector
	io.Closer
	StartupHook
}

// Listener is the basic interface anything that listens for new metrics must implement
type Listener interface {
	sfxclient.Collector
	io.Closer
	HealthChecker
}

// HealthChecker interface is anything that exports a healthcheck that would need to be invalidated on graceful shutdown
type HealthChecker interface {
	CloseHealthCheck()
}

// StartupHook interface allows a forwarder to present a callback after startup if it needs to do something that requires a fully running gateway
type StartupHook interface {
	StartupFinished() error
}

// Pipeline returns the number of items still in flight that need to be drained
type Pipeline interface {
	Pipeline() int64
}

// UneventfulForwarder converts a datapoint only forwarder into a datapoint/event forwarder
type UneventfulForwarder struct {
	DatapointForwarder
}

// StartupFinished is to be called after startup is finished
func (u *UneventfulForwarder) StartupFinished() error {
	return nil
}

// AddEvents does nothing and returns nil
func (u *UneventfulForwarder) AddEvents(ctx context.Context, events []*event.Event) error {
	return nil
}

// AddSpans does nothing and returns nil
func (u *UneventfulForwarder) AddSpans(ctx context.Context, events []*trace.Span) error {
	return nil
}

// Pipeline returns zero since UneventfulForwarder doesn't have it's own buffer
func (u *UneventfulForwarder) Pipeline() int64 {
	return 0
}

// ListenerDims are the common stat dimensions we expect on listener protocols
func ListenerDims(name string, typ string) map[string]string {
	return map[string]string{
		"location": "listener",
		"name":     name,
		"type":     typ,
	}
}

// ForwarderDims are the common stat dimensions we expect on forwarder protocols
func ForwarderDims(name string, typ string) map[string]string {
	return map[string]string{
		"location": "forwarder",
		"name":     name,
		"type":     typ,
	}
}

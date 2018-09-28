package tracing

import (
	"context"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/trace"
	"github.com/signalfx/metricproxy/protocol/signalfx"
)

type traceOnlySink struct {
	trace.Sink
}

var _ signalfx.Sink = &traceOnlySink{}

func (t *traceOnlySink) AddDatapoints(ctx context.Context, points []*datapoint.Datapoint) error {
	panic("Should not receive datapoints")
}

func (t *traceOnlySink) AddEvents(ctx context.Context, events []*event.Event) error {
	panic("Should not receive events")
}

type traceSinkFuncWrapper func(ctx context.Context, spans []*trace.Span) error

func (t traceSinkFuncWrapper) AddSpans(ctx context.Context, spans []*trace.Span) error {
	return t(ctx, spans)
}

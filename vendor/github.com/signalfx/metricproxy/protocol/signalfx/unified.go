package signalfx

import (
	"context"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/datapoint/dpsink"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/trace"
)

// Sink is a dpsink and trace.sink
type Sink interface {
	dpsink.Sink
	trace.Sink
}

// NextSink is a special case of a sink that forwards to another sink
type NextSink interface {
	AddDatapoints(ctx context.Context, points []*datapoint.Datapoint, next Sink) error
	AddEvents(ctx context.Context, events []*event.Event, next Sink) error
	AddSpans(ctx context.Context, spans []*trace.Span, next Sink) error
}

// A MiddlewareConstructor is used by FromChain to chain together a bunch of sinks that forward to each other
type MiddlewareConstructor func(sendTo Sink) Sink

// NextWrap wraps a NextSink to make it usable by MiddlewareConstructor
func NextWrap(wrapping NextSink) MiddlewareConstructor {
	return func(sendTo Sink) Sink {
		x := &nextWrapped{
			forwardTo: sendTo,
			wrapping:  wrapping,
		}
		return x
	}
}

// FromChain creates an endpoint Sink that sends calls between multiple middlewares for things like
// counting points in between.
func FromChain(endSink Sink, sinks ...MiddlewareConstructor) Sink {
	for i := len(sinks) - 1; i >= 0; i-- {
		endSink = sinks[i](endSink)
	}
	return endSink
}

type nextWrapped struct {
	forwardTo Sink
	wrapping  NextSink
}

func (n *nextWrapped) AddDatapoints(ctx context.Context, points []*datapoint.Datapoint) error {
	return n.wrapping.AddDatapoints(ctx, points, n.forwardTo)
}

func (n *nextWrapped) AddEvents(ctx context.Context, events []*event.Event) error {
	return n.wrapping.AddEvents(ctx, events, n.forwardTo)
}

func (n *nextWrapped) AddSpans(ctx context.Context, spans []*trace.Span) error {
	return n.wrapping.AddSpans(ctx, spans, n.forwardTo)
}

// IncludingDimensions returns a sink that wraps another sink adding dims to each datapoint and
// event
func IncludingDimensions(dims map[string]string, sink Sink) Sink {
	if len(dims) == 0 {
		return sink
	}
	return NextWrap(&WithDimensions{
		Dimensions: dims,
	})(sink)
}

// WithDimensions adds dimensions on top of the datapoints of a collector
type WithDimensions struct {
	Dimensions map[string]string
}

func (w *WithDimensions) appendDimensions(dps []*datapoint.Datapoint) []*datapoint.Datapoint {
	if len(w.Dimensions) == 0 {
		return dps
	}
	for _, dp := range dps {
		dp.Dimensions = datapoint.AddMaps(dp.Dimensions, w.Dimensions)
	}
	return dps
}

func (w *WithDimensions) appendDimensionsEvents(events []*event.Event) []*event.Event {
	if len(w.Dimensions) == 0 {
		return events
	}
	for _, e := range events {
		e.Dimensions = datapoint.AddMaps(e.Dimensions, w.Dimensions)
	}
	return events
}

// AddDatapoints calls next() including the wrapped dimensions on each point
func (w *WithDimensions) AddDatapoints(ctx context.Context, points []*datapoint.Datapoint, next Sink) error {
	return next.AddDatapoints(ctx, w.appendDimensions(points))
}

// AddEvents calls next() including the wrapped dimensions on each event
func (w *WithDimensions) AddEvents(ctx context.Context, events []*event.Event, next Sink) error {
	return next.AddEvents(ctx, w.appendDimensionsEvents(events))
}

// AddSpans calls next() is a pass-through
func (w *WithDimensions) AddSpans(ctx context.Context, spans []*trace.Span, next Sink) error {
	return next.AddSpans(ctx, spans)
}

type almostNextSink interface {
	dpsink.NextSink
	trace.NextSink
}

type ans struct {
	asink almostNextSink
}

func (a *ans) AddDatapoints(ctx context.Context, points []*datapoint.Datapoint, next Sink) error {
	return a.asink.AddDatapoints(ctx, points, next)
}

func (a *ans) AddEvents(ctx context.Context, events []*event.Event, next Sink) error {
	return a.asink.AddEvents(ctx, events, next)
}

func (a *ans) AddSpans(ctx context.Context, spans []*trace.Span, next Sink) error {
	return a.asink.AddSpans(ctx, spans, next)
}

// UnifyNextSinkWrap converts the combination of a dpsink.NextSink and a trace.NextSink into a signalfx.NextSink
func UnifyNextSinkWrap(s almostNextSink) NextSink {
	return &ans{
		asink: s,
	}
}

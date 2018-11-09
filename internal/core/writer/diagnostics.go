package writer

import (
	"fmt"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
)

func (sw *SignalFxWriter) averageDPM() uint {
	minutesActive := time.Since(sw.startTime).Minutes()
	return uint(float64(sw.dpsSent) / minutesActive)
}

// DiagnosticText outputs a string that describes the state of the writer to a
// human.
func (sw *SignalFxWriter) DiagnosticText() string {
	return fmt.Sprintf(
		"Writer Status:\n"+
			"Global Dims:                %s\n"+
			"Host ID Dims:               %s\n"+
			"Average DPM:                %d\n"+
			"DPs Sent:                   %d\n"+
			"Events Sent:                %d\n"+
			"DPs In Flight:              %d\n"+
			"DP Requests Active:         %d\n"+
			"Trace spans In Flight:      %d\n"+
			"Trace Span Requests Active: %d\n"+
			"Events Buffered:            %d\n"+
			"DPs Channel (len/cap) :     %d/%d\n"+
			"Events Channel (len/cap):   %d/%d\n",
		sw.conf.GlobalDimensions,
		sw.hostIDDims,
		sw.averageDPM(),
		sw.dpsSent,
		sw.eventsSent,
		sw.dpsInFlight,
		sw.dpRequestsActive,
		sw.traceSpansInFlight,
		sw.traceSpanRequestsActive,
		len(sw.eventBuffer),
		len(sw.dpChan),
		cap(sw.dpChan),
		len(sw.eventChan),
		cap(sw.eventChan))
}

// InternalMetrics returns a set of metrics showing how the writer is currently
// doing.
func (sw *SignalFxWriter) InternalMetrics() []*datapoint.Datapoint {
	return append([]*datapoint.Datapoint{
		sfxclient.Cumulative("sfxagent.datapoints_sent", nil, int64(sw.dpsSent)),
		sfxclient.Cumulative("sfxagent.events_sent", nil, int64(sw.eventsSent)),
		sfxclient.Cumulative("sfxagent.dim_prop_sets_sent", nil, int64(sw.dimPropClient.TotalPropUpdates)),
		sfxclient.Gauge("sfxagent.datapoints_buffered", nil, int64(len(sw.dpChan))),
		sfxclient.Gauge("sfxagent.datapoints_in_flight", nil, sw.dpsInFlight),
		sfxclient.Gauge("sfxagent.datapoint_requests_active", nil, sw.dpRequestsActive),
		sfxclient.Gauge("sfxagent.events_buffered", nil, int64(len(sw.eventBuffer))),
		sfxclient.Cumulative("sfxagent.trace_spans_sent", nil, int64(sw.traceSpansSent)),
		sfxclient.Cumulative("sfxagent.trace_spans_dropped", nil, int64(sw.traceSpansDropped)),
		sfxclient.Gauge("sfxagent.trace_spans_buffered", nil, int64(len(sw.spanChan))),
		sfxclient.Gauge("sfxagent.trace_spans_in_flight", nil, sw.traceSpansInFlight),
		sfxclient.Gauge("sfxagent.trace_span_requests_active", nil, sw.traceSpanRequestsActive),
	}, sw.serviceTracker.InternalMetrics()...)
}

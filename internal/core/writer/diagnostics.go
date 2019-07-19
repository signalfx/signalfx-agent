package writer

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

// Call this in a goroutine to maintain a moving window average DPM, EPM, and
// SPM, updated every 10 seconds.
func (sw *SignalFxWriter) maintainLastMinuteActivity() {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	var dpSamples [6]int64
	var dpFailedSamples [6]int64
	var eventSamples [6]int64
	var spanSamples [6]int64
	idx := 0
	for {
		select {
		case <-sw.ctx.Done():
			return
		case <-t.C:
			sw.datapointsLastMinute = atomic.LoadInt64(&sw.dpsSent) - dpSamples[idx]
			dpSamples[idx] += sw.datapointsLastMinute

			sw.datapointsFailedLastMinute = atomic.LoadInt64(&sw.dpsFailedToSend) - dpFailedSamples[idx]
			dpFailedSamples[idx] += sw.datapointsFailedLastMinute

			sw.eventsLastMinute = atomic.LoadInt64(&sw.eventsSent) - eventSamples[idx]
			eventSamples[idx] += sw.eventsLastMinute

			sw.spansLastMinute = atomic.LoadInt64(&sw.traceSpansSent) - spanSamples[idx]
			spanSamples[idx] += sw.spansLastMinute

			idx = (idx + 1) % 6
		}
	}
}

// DiagnosticText outputs a string that describes the state of the writer to a
// human.
func (sw *SignalFxWriter) DiagnosticText() string {
	return fmt.Sprintf(
		"Global Dimensions:                %s\n"+
			"Datapoints sent (last minute):    %d\n"+
			"Datapoints failed (last minute):  %d\n"+
			"Events Sent (last minute):        %d\n"+
			"Trace Spans Sent (last minute):   %d",
		utils.FormatStringMapCompact(utils.MergeStringMaps(sw.conf.GlobalDimensions, sw.hostIDDims)),
		sw.datapointsLastMinute,
		sw.datapointsFailedLastMinute,
		sw.eventsLastMinute,
		sw.spansLastMinute)
}

// InternalMetrics returns a set of metrics showing how the writer is currently
// doing.
func (sw *SignalFxWriter) InternalMetrics() []*datapoint.Datapoint {
	return append(append([]*datapoint.Datapoint{
		sfxclient.CumulativeP("sfxagent.datapoints_sent", nil, &sw.dpsSent),
		sfxclient.CumulativeP("sfxagent.datapoints_produced", nil, &sw.dpsReceived),
		sfxclient.CumulativeP("sfxagent.datapoints_filtered", nil, &sw.dpsFiltered),
		sfxclient.CumulativeP("sfxagent.datapoints_failed", nil, &sw.dpsFailedToSend),
		sfxclient.CumulativeP("sfxagent.events_sent", nil, &sw.eventsSent),
		sfxclient.Gauge("sfxagent.datapoint_channel_len", nil, int64(len(sw.dpChan))),
		sfxclient.Gauge("sfxagent.datapoints_in_flight", nil, atomic.LoadInt64(&sw.dpsInFlight)),
		sfxclient.Gauge("sfxagent.datapoints_waiting", nil, atomic.LoadInt64(&sw.dpsWaiting)),
		sfxclient.Gauge("sfxagent.datapoint_requests_active", nil, atomic.LoadInt64(&sw.dpRequestsActive)),
		sfxclient.Gauge("sfxagent.events_buffered", nil, int64(len(sw.eventBuffer))),
		sfxclient.CumulativeP("sfxagent.trace_spans_sent", nil, &sw.traceSpansSent),
		sfxclient.CumulativeP("sfxagent.trace_spans_failed", nil, &sw.traceSpansFailedToSend),
		sfxclient.CumulativeP("sfxagent.trace_spans_dropped", nil, &sw.traceSpansDropped),
		sfxclient.Gauge("sfxagent.trace_spans_buffered", nil, int64(len(sw.spanChan))),
		sfxclient.Gauge("sfxagent.trace_spans_in_flight", nil, sw.traceSpansInFlight),
		sfxclient.Gauge("sfxagent.trace_span_requests_active", nil, sw.traceSpanRequestsActive),
	}, sw.serviceTracker.InternalMetrics()...), sw.dimPropClient.InternalMetrics()...)
}

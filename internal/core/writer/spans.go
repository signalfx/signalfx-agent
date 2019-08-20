package writer

import (
	"context"
	"os"
	"sync/atomic"

	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/trace"
	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/core/common/dpmeta"
	"github.com/signalfx/signalfx-agent/internal/core/writer/tracetracker"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

func (sw *SignalFxWriter) listenForTraceSpans() {
	reqSema := make(chan struct{}, sw.conf.MaxRequests)

	// The only reason this is on the struct and not a local var is so we can
	// easily get diagnostic metrics from it
	sw.serviceTracker = sw.startGeneratingHostCorrelationMetrics()
	shedRequests := make(chan struct{})
	shedCompleted := make(chan struct{})

	for {
		select {
		case <-sw.ctx.Done():
			return

		case span := <-sw.spanChan:
			buf := append(*sw.spanBufferPool.Get().(*[]*trace.Span), span)
			buf = sw.drainSpanChan(buf)

			for i := range buf {
				sw.preprocessSpan(buf[i])
			}

			if *sw.conf.SendTraceHostCorrelationMetrics {
				sw.serviceTracker.AddSpans(sw.ctx, buf)
			}

			if atomic.LoadInt64(&sw.traceSpansInFlight) > int64(sw.conf.MaxTraceSpansInFlight) {
				// Attempt to prioritize new spans over old spans if we get in
				// a situation where we need to drop spans.  If we can't shed
				// any pending spans (e.g. because they are in the middle of
				// being sent) then drop the current set of spans we are
				// processing.  This is an imperfect process that could
				// unnecessarily shed spans if the count of spans in flight
				// drops below the threshold between the above check and the
				// shedding, but it does guarantee that the number of pending
				// spans doesn't exceed MaxTraceSpansInFlight +
				// TraceSpanMaxBatchSize.
				if !sw.attemptToShedPendingSpans(shedRequests, shedCompleted) {
					log.Warnf("Dropping %d new trace spans due to excess trace spans in flight", len(buf))
					atomic.AddInt64(&sw.traceSpansDropped, int64(len(buf)))
					continue
				}
			}

			atomic.AddInt64(&sw.traceSpansInFlight, int64(len(buf)))

			go func(spanBuf []*trace.Span) {
				defer func() {
					spanBuf = spanBuf[:0]
					sw.spanBufferPool.Put(&spanBuf)
				}()

				// Wait if there are more than the max outstanding requests,
				// but respond to requests to shed outstandand requests and the
				// writer shutdown.
				select {
				case <-sw.ctx.Done():
					atomic.AddInt64(&sw.traceSpansInFlight, -int64(len(spanBuf)))
					return
				case <-shedRequests:
					atomic.AddInt64(&sw.traceSpansDropped, int64(len(spanBuf)))
					log.Warnf("Aborting pending trace span request with %d spans "+
						"due to excess trace spans in flight", len(spanBuf))
					atomic.AddInt64(&sw.traceSpansInFlight, -int64(len(spanBuf)))
					shedCompleted <- struct{}{}
					return
				case reqSema <- struct{}{}:
					break
				}

				// Don't put this above because shed requests need to control
				// the order that they decrement this and notify completion.
				defer atomic.AddInt64(&sw.traceSpansInFlight, -int64(len(spanBuf)))

				atomic.AddInt64(&sw.traceSpanRequestsActive, 1)
				// This sends synchonously
				err := sw.client.AddSpans(context.Background(), spanBuf)
				<-reqSema
				atomic.AddInt64(&sw.traceSpanRequestsActive, -1)

				if err != nil {
					log.WithFields(log.Fields{
						"error": err,
					}).Errorf("Error shipping %d trace spans to SignalFx", len(spanBuf))
					atomic.AddInt64(&sw.traceSpansFailedToSend, int64(len(spanBuf)))
					// If there is an error sending spans then just forget about them.
					return
				}
				atomic.AddInt64(&sw.traceSpansSent, int64(len(spanBuf)))
				log.Debugf("Sent %d trace spans to SignalFx", len(spanBuf))
			}(buf)
		}
	}
}

func (sw *SignalFxWriter) attemptToShedPendingSpans(shedRequests chan struct{}, shedCompleted chan struct{}) bool {
	for {
		select {
		case shedRequests <- struct{}{}:
			// There is always a 1:1 correspondance between the request and
			// completion signal.  This guarantees that traceSpansInFlight is
			// decremented for the shed spans.
			<-shedCompleted
			if atomic.LoadInt64(&sw.traceSpansInFlight) < int64(sw.conf.MaxTraceSpansInFlight) {
				return true
			}
		default:
			// No outstanding requests are available to shed so nothing to do
			return false
		}
	}
}

func (sw *SignalFxWriter) drainSpanChan(buf []*trace.Span) []*trace.Span {
	for {
		select {
		case span := <-sw.spanChan:
			buf = append(buf, span)
			if len(buf) >= sw.conf.TraceSpanMaxBatchSize {
				return buf
			}
		default:
			return buf
		}
	}
}

func (sw *SignalFxWriter) preprocessSpan(span *trace.Span) {
	// Some spans aren't really specific to the host they are running
	// on and shouldn't have any host-specific tags.  This is indicated by a
	// special tag key (value is irrelevant).
	if _, ok := span.Tags[dpmeta.NotHostSpecificMeta]; !ok {
		span.Tags = sw.addhostIDFields(span.Tags)
	} else {
		// Get rid of the tag so it doesn't pass through to the backend
		delete(span.Tags, dpmeta.NotHostSpecificMeta)
	}

	// adding smart agent version as a tag
	span.Tags[constants.SmartAgentVersionTagKey] = os.Getenv(constants.AgentVersionEnvVar)
	if sw.conf.LogTraceSpans {
		log.Debugf("Sending trace span:\n%s", spew.Sdump(span))
	}
}

func (sw *SignalFxWriter) startGeneratingHostCorrelationMetrics() *tracetracker.ActiveServiceTracker {
	tracker := tracetracker.New(sw.conf.StaleServiceTimeout, func(dp *datapoint.Datapoint) {
		// Immediately send correlation datapoints when we first see a service
		sw.dpChan <- dp
	})

	// Send the correlation datapoints at a regular interval to keep the
	// service live on the backend.
	utils.RunOnInterval(sw.ctx, func() {
		for _, dp := range tracker.CorrelationDatapoints() {
			sw.dpChan <- dp
		}
	}, sw.conf.TraceHostCorrelationMetricsInterval)

	return tracker
}

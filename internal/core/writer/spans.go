package writer

import (
	"context"
	"encoding/json"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/trace"
	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/core/common/dpmeta"
	"github.com/signalfx/signalfx-agent/internal/core/writer/tracetracker"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

func (sw *SignalFxWriter) sendSpans(ctx context.Context, spans []*trace.Span) error {
	if *sw.conf.SendTraceHostCorrelationMetrics {
		sw.serviceTracker.AddSpans(sw.ctx, spans)
	}

	// This sends synchonously
	return sw.client.AddSpans(context.Background(), spans)
}

func (sw *SignalFxWriter) preprocessSpan(span *trace.Span) bool {
	// Some spans aren't really specific to the host they are running
	// on and shouldn't have any host-specific tags.  This is indicated by a
	// special tag key (value is irrelevant).
	if _, ok := span.Tags[dpmeta.NotHostSpecificMeta]; !ok {
		span.Tags = sw.addhostIDFields(span.Tags)
	} else {
		// Get rid of the tag so it doesn't pass through to the backend
		delete(span.Tags, dpmeta.NotHostSpecificMeta)
	}

	sw.spanSourceTracker.AddSourceTagsToSpan(span)

	// adding smart agent version as a tag
	span.Tags["signalfx.smartagent.version"] = constants.Version
	if sw.conf.LogTraceSpans {
		jsonEncoded, _ := json.Marshal(span)
		log.Infof("Sending trace span:\n%s", string(jsonEncoded))
	}

	return true
}

func (sw *SignalFxWriter) startGeneratingHostCorrelationMetrics() *tracetracker.ActiveServiceTracker {
	tracker := tracetracker.New(sw.conf.StaleServiceTimeout, func(dp *datapoint.Datapoint) {
		// Immediately send correlation datapoints when we first see a service
		sw.dpChan <- []*datapoint.Datapoint{dp}
	})

	// Send the correlation datapoints at a regular interval to keep the
	// service live on the backend.
	utils.RunOnInterval(sw.ctx, func() {
		for _, dp := range tracker.CorrelationDatapoints() {
			sw.dpChan <- []*datapoint.Datapoint{dp}
		}
	}, sw.conf.TraceHostCorrelationMetricsInterval)

	return tracker
}

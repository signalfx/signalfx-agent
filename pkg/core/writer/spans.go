package writer

import (
	"context"
	"encoding/json"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/trace"
	"github.com/signalfx/signalfx-agent/pkg/core/common/constants"
	"github.com/signalfx/signalfx-agent/pkg/core/common/dpmeta"
	"github.com/signalfx/signalfx-agent/pkg/core/writer/tracetracker"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	log "github.com/sirupsen/logrus"
)

func (sw *SignalFxWriter) sendSpans(ctx context.Context, spans []*trace.Span) error {
	if sw.serviceTracker != nil {
		sw.serviceTracker.AddSpans(sw.ctx, spans)
	}

	// This sends synchronously
	return sw.client.AddSpans(context.Background(), spans)
}

// Mutates span tags in place to add global span tags.  Also
// returns tags in case they were nil to begin with, so the return value should
// be assigned back to the span Tags field.
func (sw *SignalFxWriter) addGlobalSpanTags(tags map[string]string) map[string]string {
	if tags == nil {
		tags = make(map[string]string)
	}
	for name, value := range sw.conf.GlobalSpanTags {
		// If the tags are already set, don't override
		if _, ok := tags[name]; !ok {
			tags[name] = value
		}
	}
	return tags
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

	span.Tags = sw.addGlobalSpanTags(span.Tags)

	sw.spanSourceTracker.AddSourceTagsToSpan(span)

	if sw.conf.AddGlobalDimensionsAsSpanTags {
		span.Tags = sw.addGlobalDims(span.Tags)
	}

	// adding smart agent version as a tag
	span.Tags["signalfx.smartagent.version"] = constants.Version
	if sw.conf.LogTraceSpans {
		jsonEncoded, _ := json.Marshal(span)
		log.Infof("Sending trace span:\n%s", string(jsonEncoded))
	}

	return true
}

func (sw *SignalFxWriter) startHostCorrelationTracking() *tracetracker.ActiveServiceTracker {
	var sendTraceHostCorrelationMetrics bool
	if sw.conf.SendTraceHostCorrelationMetrics != nil {
		sendTraceHostCorrelationMetrics = *sw.conf.SendTraceHostCorrelationMetrics
	}

	tracker := tracetracker.New(sw.conf.StaleServiceTimeout.AsDuration(), sw.correlationClient, sw.hostIDDims, sendTraceHostCorrelationMetrics, func(dp *datapoint.Datapoint) {
		// Immediately send correlation datapoints when we first see a service
		sw.dpChan <- []*datapoint.Datapoint{dp}
	})

	// purge the active service tracker periodically
	utils.RunOnInterval(sw.ctx, func() {
		tracker.Purge()
	}, sw.conf.TraceHostCorrelationPurgeInterval.AsDuration())

	// Send the correlation datapoints at a regular interval to keep the
	// service live on the backend.
	utils.RunOnInterval(sw.ctx, func() {
		for _, dp := range tracker.CorrelationDatapoints() {
			sw.dpChan <- []*datapoint.Datapoint{dp}
		}
	}, sw.conf.TraceHostCorrelationMetricsInterval.AsDuration())

	return tracker
}

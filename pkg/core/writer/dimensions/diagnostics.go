package dimensions

import (
	"sync/atomic"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/sfxclient"
)

// InternalMetrics returns datapoints that describe the current state of the
// dimension update client
func (dc *DimensionClient) InternalMetrics() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		sfxclient.CumulativeP("sfxagent.dim_updates_started", nil, &dc.requestSender.TotalRequestsStarted),
		sfxclient.CumulativeP("sfxagent.dim_updates_completed", nil, &dc.requestSender.TotalRequestsCompleted),
		sfxclient.CumulativeP("sfxagent.dim_updates_failed", nil, &dc.requestSender.TotalRequestsFailed),
		sfxclient.Gauge("sfxagent.dim_request_senders", nil, atomic.LoadInt64(&dc.requestSender.RunningWorkers)),
		sfxclient.Gauge("sfxagent.dim_updates_currently_delayed", nil, atomic.LoadInt64(&dc.DimensionsCurrentlyDelayed)),
		sfxclient.CumulativeP("sfxagent.dim_updates_dropped", nil, &dc.TotalDimensionsDropped),
		sfxclient.CumulativeP("sfxagent.dim_updates_invalid", nil, &dc.TotalInvalidDimensions),
		sfxclient.CumulativeP("sfxagent.dim_updates_flappy_total", nil, &dc.TotalFlappyUpdates),
		sfxclient.CumulativeP("sfxagent.dim_updates_duplicates", nil, &dc.TotalDuplicates),
		// All 4xx HTTP responses that are not retried except 404 (which is retried)
		sfxclient.CumulativeP("sfxagent.dim_updates_client_errors", nil, &dc.TotalClientError4xxResponses),
		sfxclient.CumulativeP("sfxagent.dim_updates_retries", nil, &dc.TotalRetriedUpdates),
		sfxclient.Cumulative("sfxagent.dim_updates_deduplicator_size", nil, int64(dc.deduplicator.history.Len())),
	}
}

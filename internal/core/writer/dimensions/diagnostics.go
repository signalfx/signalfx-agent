package dimensions

import (
	"sync/atomic"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
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
		sfxclient.CumulativeP("sfxagent.dim_updates_flappy_total", nil, &dc.TotalFlappyUpdates),
	}
}

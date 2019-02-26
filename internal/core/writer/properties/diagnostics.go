package properties

import (
	"sync/atomic"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
)

// InternalMetrics returns datapoints that describe the current state of the
// dimension update client
func (dpc *DimensionPropertyClient) InternalMetrics() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		sfxclient.CumulativeP("sfxagent.dim_updates_started", nil, &dpc.requestSender.TotalRequestsStarted),
		sfxclient.CumulativeP("sfxagent.dim_updates_completed", nil, &dpc.requestSender.TotalRequestsCompleted),
		sfxclient.CumulativeP("sfxagent.dim_updates_failed", nil, &dpc.requestSender.TotalRequestsFailed),
		sfxclient.Gauge("sfxagent.dim_request_senders", nil, atomic.LoadInt64(&dpc.requestSender.RunningWorkers)),
		sfxclient.Gauge("sfxagent.dim_updates_currently_delayed", nil, atomic.LoadInt64(&dpc.DimensionsCurrentlyDelayed)),
		sfxclient.CumulativeP("sfxagent.dim_updates_dropped", nil, &dpc.TotalDimensionsDropped),
		sfxclient.CumulativeP("sfxagent.dim_updates_flappy_total", nil, &dpc.TotalFlappyUpdates),
	}
}

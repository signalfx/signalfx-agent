package requests

import (
	"sync/atomic"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/sfxclient"
)

// InternalMetrics returns datapoints that describe the current state of the
// dimension update client
func (rs *ReqSender) InternalMetrics() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		sfxclient.CumulativeP("sfxagent.dim_updates_started", rs.additionalDimensions, &rs.TotalRequestsStarted),
		sfxclient.CumulativeP("sfxagent.dim_updates_completed", rs.additionalDimensions, &rs.TotalRequestsCompleted),
		sfxclient.CumulativeP("sfxagent.dim_updates_failed", rs.additionalDimensions, &rs.TotalRequestsFailed),
		sfxclient.Gauge("sfxagent.dim_request_senders", rs.additionalDimensions, atomic.LoadInt64(&rs.RunningWorkers)),
	}
}

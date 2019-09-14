package haproxy

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/utils"

	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	logger "github.com/sirupsen/logrus"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Monitor for reporting HAProxy stats.
//
// In multi-process mode HAProxy reports stats/info randomly one process at a time upon query. There is no stat for
// the number of processes for proxy-related 'csv' stats. There is however, stat pid which is an index for individual
// processes. Likewise process-related 'info' stats have Process_num. The info stats do however have Nbproc which gives
// the number of processes stat. But, info stats are not available through http.
//
// This monitor assigns the pid stat and Process_num info to dimension process_num. It finds the number of running
// HAProxy processes dynamically by evaluating the maximum value of dimension process_num over time. It fetches HAProxy
// stats repeatedly if necessary in order to get stats for all processes. It fetches stats repeatedly if necessary
// within the configured timeout duration.
type Monitor struct {
	Output types.Output
	cancel context.CancelFunc
	ctx    context.Context
}

// Config for this monitor
func (m *Monitor) Configure(conf *Config) (err error) {
	m.ctx, m.cancel = context.WithCancel(context.Background())

	url, err := url.Parse(conf.URL)
	if err != nil {
		return fmt.Errorf("cannot parse url %s status. %v", conf.URL, err)
	}

	var fetch func(context.Context, *Config, int) []*datapoint.Datapoint
	switch url.Scheme {
	case "http", "https", "file":
		fetch = mergeHTTP
	case "unix":
		fetch = mergeSocket
	default:
		return fmt.Errorf("unsupported scheme: %q", url.Scheme)
	}

	numProcesses, purgeCountDown, pidCache := 1, 6, map[string]bool{}

	interval := time.Duration(conf.IntervalSeconds) * time.Second

	utils.RunOnInterval(m.ctx, func() {
		ctx, cancel := context.WithTimeout(m.ctx, interval)
		defer cancel()
		for _, dp := range fetch(ctx, conf, numProcesses) {
			dp.Dimensions["plugin"] = "haproxy"
			pidCache[dp.Dimensions["process_num"]] = true
			m.Output.SendDatapoint(dp)
		}
		// Purge stale pids unseen after count down to 0.
		if purgeCountDown--; purgeCountDown == 0 {
			for pid, hit := range pidCache {
				if !hit {
					delete(pidCache, pid)
					continue
				}
				pidCache[pid] = false
			}
			purgeCountDown = 6
		}
		// Count pids to get number of processes. Assign 1 if none seen.
		if numProcesses = len(pidCache); numProcesses == 0 {
			logger.Errorf("got no HAProxy process pid")
			numProcesses = 1
		}
	}, interval)

	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

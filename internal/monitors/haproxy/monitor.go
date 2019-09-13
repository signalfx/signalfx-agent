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
	Output  types.Output
	cancel  context.CancelFunc
	ctx     context.Context
	url     *url.URL
	proxies map[string]bool
}

// Config for this monitor
func (m *Monitor) Configure(conf *Config) (err error) {
	m.ctx, m.cancel = context.WithCancel(context.Background())

	m.url, err = url.Parse(conf.URL)
	if err != nil {
		return fmt.Errorf("cannot parse url %s status. %v", conf.URL, err)
	}

	m.proxies = map[string]bool{}
	for _, proxy := range conf.Proxies {
		m.proxies[proxy] = true
	}

	var fetch func(context.Context, *Config, int) []*datapoint.Datapoint

	switch m.url.Scheme {
	case "http", "https", "file":
		fetch = m.fetchAllHTTP
	case "unix":
		fetch = m.fetchAllSocket
	default:
		return fmt.Errorf("unsupported scheme: %q", m.url.Scheme)
	}

	numProcesses, intervalNum, reportedPids := 1, 1, map[string]int{}

	interval := time.Duration(conf.IntervalSeconds) * time.Second

	utils.RunOnInterval(m.ctx, func() {
		ctx, cancel := context.WithTimeout(m.ctx, interval)
		defer cancel()

		for _, dp := range fetch(ctx, conf, numProcesses) {
			reportedPids[dp.Dimensions["process_num"]] = intervalNum
			m.Output.SendDatapoint(dp)
		}

		// Delete pids not reported after 5 intervals and reset interval numbers.
		if intervalNum == 5 {
			intervalNum = 0
			for k, v := range reportedPids {
				if v == 0 {
					delete(reportedPids, k)
					continue
				}
				reportedPids[k] = intervalNum
			}
		}
		intervalNum++

		// Count reported pids to get number of processes. If none reported, assign 1 for next interval.
		if numProcesses = len(reportedPids); numProcesses == 0 {
			logger.Errorf("cannot find running HAProxy process(es).")
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

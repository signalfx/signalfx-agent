package haproxy

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"sync"
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

const maxRoutines = 16

// Monitor for reporting HAProxy stats.
//
// In multi-process mode HAProxy reports stats/info randomly one process at a time upon query. There is no stat for
// the number of processes for proxy-related 'csv' stats. There is however, stat pid which is an index for individual
// processes. Likewise process-related 'info' stats have Process_num. The info stats do however have Nbproc which gives
// the number of processes stat. But, info stats are not available through http.
//
// This monitor assigns the pid and Process_num stats to dimension process_num. It finds the number of processes
// dynamically by evaluating the maximum value of dimension process_num over time. It fetches HAProxy stats repeatedly
// if necessary in order to get stats for all HAProxy processes. It fetches stats repeatedly if necessary within the
// configured timeout duration.
type Monitor struct {
	Output  types.Output
	cancel  context.CancelFunc
	ctx     context.Context
	url     *url.URL
	fetch   func(context.Context, *Config, int) []*datapoint.Datapoint
	proxies map[string]bool
}

type dpsChans struct {
	chans map[string]chan []*datapoint.Datapoint
	lock  sync.Mutex
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

	switch m.url.Scheme {
	case "http", "https", "file":
		m.fetch = m.fetchAllHTTP
	case "unix":
		m.fetch = m.fetchAllSocket
	default:
		return fmt.Errorf("unsupported scheme: %q", m.url.Scheme)
	}

	numProcesses := 1

	utils.RunOnInterval(m.ctx, func() {
		// max process_num dimension value seen in datapoints
		maxProcessNum := 1
		ctx, cancel := context.WithTimeout(m.ctx, conf.Timeout)
		defer cancel()
		for _, dp := range m.fetch(ctx, conf, numProcesses) {
			if p, err := strconv.Atoi(dp.Dimensions["process_num"]); err == nil && p > maxProcessNum {
				maxProcessNum = p
			} else if err != nil {
				logger.Errorf("failed to convert into int value %s of dimension process_num. %+v", dp.Dimensions["process_num"], err)
			}
			m.Output.SendDatapoint(dp)
		}
		numProcesses = maxProcessNum
	}, time.Duration(conf.IntervalSeconds)*time.Second)
	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

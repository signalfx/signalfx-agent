package haproxy

import (
	"context"
	"fmt"
	"net/url"
	"strings"
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
// In multi-process mode, you get query results from a single randomly selected HAProxy process. Proxy statistics
// (proxy stats) can be queried through http and UNIX socket. HAProxy process information (process info) is only
// available through UNIX socket. There is no proxy stat for the total number of processes. There is however, stat pid
// which is an integer identifier assigned to individual processes. pid is bounded by the total number of processes.
// It is not the usual OS assigned process identifier. Process info has Process_num which is equivalent to pid. Process
// info does however have Nbproc which gives the total number of processes. But, process info is not available through
// http.
//
// This monitor turns proxy stats and process info into metric datapoints. It finds the total number of HAProxy
// processes dynamically over time by counting the number of unique process_num in datapoints collected. process_num is
// a datapoint dimension containing the pid value for proxy stats and Process_num for process info. Datapoints for
// multiple HAProxy processes are fetched concurrently.
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

	proxies := map[string]bool{}
	for _, p := range conf.Proxies {
		switch strings.ToLower(strings.TrimSpace(p)) {
		case "frontend":
			proxies["FRONTEND"] = true
		case "backend":
			proxies["BACKEND"] = true
		default:
			proxies[p] = true
		}
	}

	var fetch func(context.Context, *Config, int, map[string]bool) []*datapoint.Datapoint

	switch url.Scheme {
	case "http", "https", "file":
		fetch = fetchHTTP
	case "unix":
		fetch = fetchSocket
	default:
		return fmt.Errorf("unsupported scheme:%q", url.Scheme)
	}

	// Map for caching process numbers in intervals.
	pCache := map[string]bool{}
	// Number of processes. Equal to length of non-empty cache.
	numP := 1
	// Number of intervals before refreshing pCache.
	refreshCountDown := 30

	interval := time.Duration(conf.IntervalSeconds) * time.Second

	utils.RunOnInterval(m.ctx, func() {
		ctx, cancel := context.WithTimeout(m.ctx, interval)
		defer cancel()
		for _, dp := range fetch(ctx, conf, numP, proxies) {
			dp.Dimensions["plugin"] = "haproxy"
			pCache[dp.Dimensions["process_num"]] = true
			m.Output.SendDatapoint(dp)
		}
		if refreshCountDown--; refreshCountDown == 0 {
			for pNum, hit := range pCache {
				if !hit {
					delete(pCache, pNum)
					continue
				}
				pCache[pNum] = false
			}
			refreshCountDown = 30
		}
		logger.Debugf("Discovered %d HAProxy processes in this interval", len(pCache))
		if numP = len(pCache); numP == 0 {
			errmsg := "all"
			if len(conf.Proxies) > 0 {
				errmsg = fmt.Sprintf("%+v", conf.Proxies)
			}
			logger.Errorf("Failed to create datapoints. Monitored proxies: %s", errmsg)
			logger.Errorf("Discovered %d HAProxy processes in this interval. Run in debug mode to see the number of discovered processes at each intervals", len(pCache))
			numP = 1
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

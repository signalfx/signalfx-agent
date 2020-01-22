package haproxy

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/utils"

	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Monitor for reporting HAProxy stats.
type Monitor struct {
	Output types.Output
	cancel context.CancelFunc
	ctx    context.Context
}

// Map of proxies to monitor
type proxies map[string]bool

// Config for this monitor
func (m *Monitor) Configure(conf *Config) (err error) {
	m.ctx, m.cancel = context.WithCancel(context.Background())
	url, err := url.Parse(conf.ScrapeURL())
	if err != nil {
		return fmt.Errorf("cannot parse url %s status. %v", conf.ScrapeURL(), err)
	}
	pxs := proxies{}
	for _, p := range conf.Proxies {
		switch strings.ToLower(strings.TrimSpace(p)) {
		case "frontend":
			pxs["FRONTEND"] = true
		case "backend":
			pxs["BACKEND"] = true
		default:
			pxs[p] = true
		}
	}
	type funcs []func(*Config, proxies) []*datapoint.Datapoint
	var fetchFuncs funcs
	switch url.Scheme {
	case "http", "https", "file":
		fetchFuncs = funcs{statsHTTP}
	case "unix":
		fetchFuncs = funcs{statsSocket, infoSocket}
	default:
		return fmt.Errorf("unsupported url scheme:%q", url.Scheme)
	}
	interval := time.Duration(conf.IntervalSeconds) * time.Second
	utils.RunOnInterval(m.ctx, func() {
		var wg sync.WaitGroup

		for _, fn := range fetchFuncs {
			fn := fn
			wg.Add(1)
			go func() {
				dps := fn(conf, pxs)
				for i := range dps {
					dps[i].Dimensions["plugin"] = "haproxy"
				}
				m.Output.SendDatapoints(dps...)
				wg.Done()
			}()
		}

		wg.Wait()
	}, interval)
	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

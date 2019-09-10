package haproxy

import (
	"context"
	"net/url"

	"github.com/signalfx/golib/datapoint"
	logger "github.com/sirupsen/logrus"

	"time"

	"github.com/signalfx/signalfx-agent/internal/utils"

	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Monitor for Prometheus server metrics Exporter
type Monitor struct {
	Output types.Output
	cancel context.CancelFunc
	ctx    context.Context
}

// Configure the haproxy monitor
func (m *Monitor) Configure(conf *Config) (err error) {
	m.ctx, m.cancel = context.WithCancel(context.Background())
	proxiesToMonitor := map[string]bool{}
	for _, proxy := range conf.ProxiesToMonitor {
		proxiesToMonitor[proxy] = true
	}
	utils.RunOnInterval(m.ctx, func() {
		for _, dp := range m.getDatapoints(conf, proxiesToMonitor) {
			m.Output.SendDatapoint(dp)
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)
	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

func (m *Monitor) getDatapoints(conf *Config, proxiesToMonitor map[string]bool) []*datapoint.Datapoint {
	u, err := url.Parse(conf.URL)
	if err != nil {
		logger.Errorf("cannot parse url %s status. %v", conf.URL, err)
		return nil
	}
	switch u.Scheme {
	case "http", "https", "file":
		body, err := csvReader(conf)
		if err != nil {
			logger.Errorf("can't scrape HAProxy: %v", err)
			return nil
		}
		return newStatPageDatapoints(body, proxiesToMonitor)
	case "unix":
		showStatBody, err := commandReader(u, "show stat\n", conf.Timeout)
		if err != nil {
			logger.Errorf("can't scrape HAProxy: %v", err)
		}
		showInfoBody, err := commandReader(u, "show info\n", conf.Timeout)
		if err != nil {
			logger.Errorf("can't scrape HAProxy: %v", err)
			return nil
		}
		return append(newShowStatCommandDatapoints(showStatBody, proxiesToMonitor), newShowInfoCommandDatapoints(showInfoBody)...)
	default:
		logger.Errorf("unsupported scheme: %q", u.Scheme)
		return nil
	}
}

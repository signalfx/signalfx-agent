package haproxy

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/signalfx/golib/datapoint"
	logger "github.com/sirupsen/logrus"

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
	getDps func(*Config) []*datapoint.Datapoint
}

// Configure the haproxy monitor
func (m *Monitor) Configure(conf *Config) (err error) {
	m.ctx, m.cancel = context.WithCancel(context.Background())
	u, err := url.Parse(conf.URL)
	if err != nil {
		return fmt.Errorf("cannot parse url %s status. %v", conf.URL, err)
	}
	proxies := map[string]bool{}
	for _, proxy := range conf.Proxies {
		proxies[proxy] = true
	}
	switch u.Scheme {
	case "http", "https", "file":
		m.getDps = func(conf *Config) []*datapoint.Datapoint {
			return newStatsPageDps(conf, proxies)
		}
	case "unix":
		m.getDps = func(conf *Config) []*datapoint.Datapoint {
			return append(newStatsCmdDps(u, conf.Timeout, proxies), newInfoDps(u, conf.Timeout)...)
		}
	default:
		logger.Errorf("unsupported scheme: %q", u.Scheme)
		return nil
	}
	utils.RunOnInterval(m.ctx, func() {
		for _, dp := range m.getDps(conf) {
			dp.Dimensions["plugin"] = "haproxy"
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

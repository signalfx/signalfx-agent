package internalmetrics

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/meta"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/network/simpleserver"
	log "github.com/sirupsen/logrus"
)

const (
	monitorType = "internal-metrics"
)

// MONITOR(internal-metrics): Emits metrics about the internal state of the
// agent.  Useful for debugging performance issues with the agent and to ensure
// the agent isn't overloaded.

// CUMULATIVE(sfxagent.datapoints_sent): The total number of datapoints sent by
// the agent since it last started

// CUMULATIVE(sfxagent.events_sent): The total number of events sent by the
// agent since it last started

// GAUGE(sfxagent.datapoints_buffered): The total number of datapoints that
// have been emitted by monitors but have yet to be sent to SignalFx

// GAUGE(sfxagent.events_buffered): The total number of events that have been
// emitted by monitors but have yet to be sent to SignalFx

// GAUGE(sfxagent.active_monitors): The total number of monitor instances
// actively working

// GAUGE(sfxagent.configured_monitors): The total number of monitor
// configurations

// GAUGE(sfxagent.discovered_endpoints): The number of discovered service
// endpoints.  This includes endpoints that do not have any matching monitor
// configuration discovery rule.

// GAUGE(sfxagent.active_observers): The number of observers configured and
// running

// Config for internal metric monitoring
type Config struct {
	config.MonitorConfig
}

// Monitor for collecting internal metrics from the simple server that dumps
// them.
type Monitor struct {
	Output    types.Output
	AgentMeta *meta.AgentMeta
	cancel    func()
}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Configure and kick off internal metric collection
func (m *Monitor) Configure(conf *Config) error {
	m.Shutdown()

	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())
	utils.RunOnInterval(ctx, func() {
		c, err := simpleserver.Dial(m.AgentMeta.InternalMetricsServerPath)
		if err != nil {
			log.WithFields(log.Fields{
				"error":       err,
				"monitorType": monitorType,
				"path":        m.AgentMeta.InternalMetricsServerPath,
			}).Error("Could not connect to internal metric server")
			return
		}

		c.SetReadDeadline(time.Now().Add(5 * time.Second))
		jsonIn, err := ioutil.ReadAll(c)
		c.Close()
		if err != nil {
			log.WithFields(log.Fields{
				"error":       err,
				"monitorType": monitorType,
				"path":        m.AgentMeta.InternalMetricsServerPath,
			}).Error("Could not read metrics from internal metric server")
			return
		}

		dps := make([]*datapoint.Datapoint, 0)
		err = json.Unmarshal(jsonIn, &dps)

		for _, dp := range dps {
			m.Output.SendDatapoint(dp)
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown the internal metric collection
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

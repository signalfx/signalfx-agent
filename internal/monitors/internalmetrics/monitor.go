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
//
// ```yaml
// monitors:
//   - type: internal-metrics
// ```
//

// CUMULATIVE(sfxagent.datapoints_sent): The total number of datapoints sent by
// the agent since it last started

// CUMULATIVE(sfxagent.events_sent): The total number of events sent by the
// agent since it last started

// GAUGE(sfxagent.datapoints_buffered): The total number of datapoints that
// have been emitted by monitors but have yet to be processed by the writer

// GAUGE(sfxagent.datapoints_in_flight): The total number of datapoints that
// have been accepted by the writer but still lack confirmation from ingest
// that they have been received.

// GAUGE(sfxagent.datapoint_requests_active): The total number of outstanding
// requests to ingest currently active.

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

// CUMULATIVE(sfxagent.go_total_alloc): Total number of bytes allocated to the
// heap throughout the lifetime of the agent

// CUMULATIVE(sfxagent.go_mallocs): Total number of heap objects allocated
// throughout the lifetime of the agent

// CUMULATIVE(sfxagent.go_frees): Total number of heap objects freed
// throughout the lifetime of the agent

// GAUGE(sfxagent.go_heap_alloc): Bytes of live heap memory (memory that has
// been allocated but not freed)

// GAUGE(sfxagent.go_heap_idle): Bytes of memory that consist of idle spans
// (that is, completely empty spans of memory)

// GAUGE(sfxagent.go_heap_released): Bytes of memory that have been returned to
// the OS.  This is quite often 0.  `sfxagent.go_heap_idle -
// sfxagent.go_heap_release` is the memory that Go is retaining for future heap
// allocations.

// GAUGE(sfxagent.go_heap_sys): Virtual memory size in bytes of the agent.  This
// will generally reflect the largest heap size the agent has ever had in its
// lifetime.

// GAUGE(sfxagent.go_heap_inuse): Size in bytes of in use spans

// GAUGE(sfxagent.go_stack_inuse): Size in bytes of spans that have at least
// one goroutine stack in them

// GAUGE(sfxagent.go_next_gc): The target heap size -- GC tries to keep the
// heap smaller than this

// GAUGE(sfxagent.go_num_gc): The number of GC cycles that have happened in the
// agent since it started

// GAUGE(sfxgent.go_num_goroutine): Number of goroutines in the agent

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

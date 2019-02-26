package internalmetrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/meta"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

const (
	monitorType = "internal-metrics"
)

// MONITOR(internal-metrics): Emits metrics about the internal state of the
// agent.  Useful for debugging performance issues with the agent and to ensure
// the agent isn't overloaded.
//
// This can also scrape any HTTP endpoint that exposes metrics as a JSON array
// containing JSON-formatted SignalFx datapoint objects.  It is roughly
// analogous to the `prometheus-exporter` monitor except for SignalFx
// datapoints.
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

// CUMULATIVE(sfxagent.dim_updates_started): Total number of dimension property
// updates requests started, but not necessarily completed or failed.

// CUMULATIVE(sfxagent.dim_updates_completed): Total number of dimension
// property updates successfully completed

// CUMULATIVE(sfxagent.dim_updates_failed): Total number of dimension property
// updates that failed for some reason.  The failures should be logged.

// GAUGE(sfxagent.dim_request_senders): Current number of worker goroutines
// active that can send dimension updates.

// GAUGE(sfxagent.dim_updates_currently_delayed): Current number of dimension
// updates that are being delayed to avoid sending spurious updates due to
// flappy dimension property sets.

// CUMULATIVE(sfxagent.dim_updates_dropped): Total number of dimension property
// updates that were dropped, due to an overfull buffer of dimension updates
// pending.

// CUMULATIVE(sfxagent.dim_updates_flappy_total): Total number of dimension
// property updates that ended up replacing a dimension property set that was
// being delayed.

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
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

	// Defaults to the top-level `internalStatusHost` option
	Host string `yaml:"host"`
	// Defaults to the top-level `internalStatusPort` option
	Port uint16 `yaml:"port" noDefault:"true"`
	// The HTTP request path to use to retrieve the metrics
	Path string `yaml:"path" default:"/metrics"`
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

	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	utils.RunOnInterval(ctx, func() {
		// Derive the url each time since the AgentMeta data can change but
		// there is no notification system for it.
		host := conf.Host
		if host == "" {
			host = m.AgentMeta.InternalStatusHost
		}

		port := conf.Port
		if port == 0 {
			port = m.AgentMeta.InternalStatusPort
		}

		url := fmt.Sprintf("http://%s:%d%s", host, port, conf.Path)

		logger := log.WithFields(log.Fields{
			"monitorType": monitorType,
			"url":         url,
		})

		resp, err := client.Get(url)
		if err != nil {
			logger.WithError(err).Error("Could not connect to internal metric server")
			return
		}
		defer resp.Body.Close()

		dps := make([]*datapoint.Datapoint, 0)
		err = json.NewDecoder(resp.Body).Decode(&dps)
		if err != nil {
			logger.WithError(err).Error("Could not parse metrics from internal metric server")
			return
		}

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

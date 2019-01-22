// +build windows

package dotnet

import (
	"context"
	"time"

	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/batchemitter"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winperfcounters"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winperfcounters/perfhelper"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

var metricMap = map[string]*perfhelper.MetricMapper{
	"dotnet_clr_exceptions.#_of_exceps_thrown_/_sec":           {Name: "net_clr_exceptions.num_exceps_thrown_sec"},
	"dotnet_clr_locksandthreads.#_of_current_logical_threads":  {Name: "net_clr_locksandthreads.num_of_current_logical_threads"},
	"dotnet_clr_locksandthreads.#_of_current_physical_threads": {Name: "net_clr_locksandthreads.num_of_current_physical_threads"},
	"dotnet_clr_locksandthreads.contention_rate_/_sec":         {Name: "net_clr_locksandthreads.contention_rate_sec"},
	"dotnet_clr_locksandthreads.current_queue_length":          {Name: "net_clr_locksandthreads.current_queue_length"},
	"dotnet_clr_memory.#_bytes_in_all_heaps":                   {Name: "net_clr_memory.num_bytes_in_all_heaps"},
	"dotnet_clr_memory.percent_time_in_gc":                     {Name: "net_clr_memory.pct_time_in_gc"},
	"dotnet_clr_memory.#_gc_handles":                           {Name: "net_clr_memory.num_gc_handles"},
	"dotnet_clr_memory.#_total_committed_bytes":                {Name: "net_clr_memory.num_total_committed_bytes"},
	"dotnet_clr_memory.#_total_reserved_bytes":                 {Name: "net_clr_memory.num_total_reserved_bytes"},
	"dotnet_clr_memory.#_of_pinned_objects":                    {Name: "net_clr_memory.num_of_pinned_objects"},
}

// Configure the monitor and kick off metric syncing
func (m *Monitor) Configure(conf *Config) error {
	perfcounterConf := &winperfcounters.Config{
		CountersRefreshInterval: conf.CountersRefreshInterval,
		PrintValid:              conf.PrintValid,
		Object: []winperfcounters.PerfCounterObj{
			{
				ObjectName: ".NET CLR Exceptions",
				Counters: []string{
					"# of exceps thrown / sec",
				},
				Instances:     []string{"*"},
				Measurement:   "dotnet_clr_exceptions",
				IncludeTotal:  true,
				WarnOnMissing: true,
			},
			{
				ObjectName: ".NET CLR LocksAndThreads",
				Counters: []string{
					"# of current logical threads",
					"# of current physical threads",
					"contention rate / sec",
					"current queue length",
				},
				Instances:     []string{"*"},
				Measurement:   "dotnet_clr_locksandthreads",
				IncludeTotal:  true,
				WarnOnMissing: true,
			},
			{
				ObjectName: ".NET CLR Memory",
				Counters: []string{
					"# bytes in all heaps",
					"% time in gc",
					"# gc handles",
					"# total committed bytes",
					"# total reserved bytes",
					"# of pinned objects",
				},
				Instances:     []string{"*"},
				Measurement:   "dotnet_clr_memory",
				IncludeTotal:  true,
				WarnOnMissing: true,
			},
		},
	}

	plugin := winperfcounters.GetPlugin(perfcounterConf)

	// create batch emitter
	emitter := batchemitter.NewEmitter(m.Output, logger)

	// create the accumulator
	ac := accumulator.NewAccumulator(emitter)

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		if err := plugin.Gather(ac); err != nil {
			logger.Error(err)
		}

		// process measurements retrieved from perf counter monitor
		for _, e := range perfhelper.ProcessMeasurements(emitter.Measurements, metricMap, m.Output.SendDatapoint, monitorType, "instance") {
			logger.Error(e.Error())
		}

		// reset batch emitter
		// NOTE: we can do this here because this emitter is on a single routine
		// if that changes, make sure you lock the mutex on the batch emitter
		emitter.Measurements = emitter.Measurements[:0]

	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

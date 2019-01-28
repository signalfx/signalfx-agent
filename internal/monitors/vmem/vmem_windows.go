// +build windows

package vmem

import (
	"context"
	"time"

	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/baseemitter"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winperfcounters"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

var metricNameMapping = map[string]string{
	"win_memory.Pages_Input_persec":  "vmpage.swap.in_per_second",
	"win_memory.Pages_Output_persec": "vmpage.swap.out_per_second",
	"win_memory.Pages_persec":        "vmpage.swap.total_per_second",
}

// Configure and run the monitor on windows
func (m *Monitor) Configure(conf *Config) (err error) {
	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	perfcounterConf := &winperfcounters.Config{
		CountersRefreshInterval: conf.CountersRefreshInterval,
		PrintValid:              conf.PrintValid,
		Object: []winperfcounters.PerfCounterObj{
			{
				// The name of a windows performance counter object
				ObjectName: "Memory",
				// The name of the counters to collect from the performance counter object
				Counters: []string{"Pages Input/sec", "Pages Output/sec", "Pages/sec"},
				// The windows performance counter instances to fetch for the performance counter object
				Instances: []string{"------"},
				// The name of the telegraf measurement that will be used as a metric name
				Measurement: "win_memory",
				// Log a warning if the perf counter object is missing
				WarnOnMissing: true,
				// Include the total instance when collecting performance counter metrics
				IncludeTotal: true,
			},
		},
	}

	plugin := winperfcounters.GetPlugin(perfcounterConf)

	// create batch emitter
	emitter := baseemitter.NewEmitter(m.Output, logger)

	// add metric map to rename metrics
	emitter.RenameMetrics(metricNameMapping)

	// Hard code the plugin name because the emitter will parse out the
	// configured measurement name as plugin and that is confusing.
	emitter.AddTag("plugin", monitorType)

	// omit instance tags from dimensions
	emitter.OmitTag("instance")

	// create the accumulator
	ac := accumulator.NewAccumulator(emitter)

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		if err := plugin.Gather(ac); err != nil {
			logger.WithError(err).Errorf("unable to gather metrics from plugin")
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

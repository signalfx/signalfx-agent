// +build windows

package aspdotnet

import (
	"context"
	"time"

	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/baseemitter"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winperfcounters"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

// Configure the monitor and kick off metric syncing
func (m *Monitor) Configure(conf *Config) error {
	perfcounterConf := &winperfcounters.Config{
		CountersRefreshInterval: conf.CountersRefreshInterval,
		PrintValid:              conf.PrintValid,
		Object: []winperfcounters.PerfCounterObj{
			{
				ObjectName: "ASP.NET",
				Counters: []string{
					"applications running",
					"application restarts",
					"requests current",
					"requests queued",
					"requests rejected",
					"worker processes running",
					"worker process restarts",
				},
				Instances:     []string{"*"},
				Measurement:   "asp_net",
				IncludeTotal:  true,
				WarnOnMissing: true,
			},
			{
				ObjectName: "ASP.NET Applications",
				Counters: []string{
					"requests failed",
					"requests/sec",
					"errors during execution",
					"errors unhandled during execution/sec",
					"errors total/sec",
					"pipeline instance count",
					"sessions active",
					"session sql server connections total",
				},
				Instances:     []string{"*"},
				Measurement:   "asp_net_applications",
				IncludeTotal:  true,
				WarnOnMissing: true,
			},
		},
	}

	plugin := winperfcounters.GetPlugin(perfcounterConf)

	// create batch emitter
	emitter := baseemitter.NewEmitter(m.Output, logger)

	// create base emitter
	emitter = baseemitter.NewEmitter(m.Output, logger)

	// set metric name replacements to match SignalFx PerfCounterReporter
	emitter.AddMetricNameTransformation(winperfcounters.NewPCRMetricNamesTransformer())

	// sanitize the instance tag associated with windows perf counter metrics
	emitter.AddMeasurementTransformation(winperfcounters.NewPCRInstanceTagTransformer())

	// create the accumulator
	ac := accumulator.NewAccumulator(emitter)

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		if err := plugin.Gather(ac); err != nil {
			logger.WithError(err).Errorf("an error occurred while gathering metrics from the plugin")
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

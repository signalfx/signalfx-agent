// +build windows

package vmem

import (
	"context"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/batchemitter"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/measurement"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winperfcounters"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

var metricNameMapping = map[string]string{
	"Pages_Input_persec":  "vmpage.swap.in_per_second",
	"Pages_Output_persec": "vmpage.swap.out_per_second",
	"Pages_persec":        "vmpage.swap.total_per_second",
}

func (m *Monitor) processMeasurement(ms *measurement.Measurement) {
	if len(ms.Fields) == 0 {
		logger.Errorf("no fields on measurement '%s'", ms.Measurement)
	}
	dimensions := map[string]string{"plugin": monitorType}
	for field, val := range ms.Fields {
		metricName, ok := metricNameMapping[field]
		if !ok {
			logger.Errorf("unable to map field '%s' to a metricname while parsing measurement '%s'",
				field, ms.Measurement)
			continue
		}

		// parse metric value
		var metricVal datapoint.Value
		var err error
		if metricVal, err = datapoint.CastMetricValue(val); err != nil {
			logger.WithError(err).Errorf("failed to cast metric value for field '%s' on measurement '%s'", field, ms.Measurement)
			continue
		}

		m.Output.SendDatapoint(datapoint.New(metricName, dimensions, metricVal, datapoint.Gauge, time.Time{}))
	}
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
	emitter := batchemitter.NewEmitter(m.Output, logger)

	// create the accumulator
	ac := accumulator.NewAccumulator(emitter)

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		if err := plugin.Gather(ac); err != nil {
			logger.Error(err)
		}

		// reset batch emitter
		// NOTE: we can do this here because this emitter is on a single routine
		// if that changes, make sure you lock the mutex on the batch emitter
		for _, ms := range emitter.Measurements {
			m.processMeasurement(ms)
		}
		emitter.Measurements = emitter.Measurements[:0]

	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

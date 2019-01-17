// +build windows

package winperfcounters

import (
	"context"
	"time"

	telegrafInputs "github.com/influxdata/telegraf/plugins/inputs"
	telegrafPlugin "github.com/influxdata/telegraf/plugins/inputs/win_perf_counters"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/baseemitter"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/ulule/deepcopier"
)

// fetch the factory used to generate the perf counter plugin
var factory = telegrafInputs.Inputs["win_perf_counters"]

// GetPlugin takes a perf counter monitor config and returns a configured perf counter plugin.
// This is used for other monitors based on perf counter that manage their own life cycle
// (i.e. system utilization, windows iis)
func GetPlugin(conf *Config) *telegrafPlugin.Win_PerfCounters {
	plugin := factory().(*telegrafPlugin.Win_PerfCounters)

	// copy top level struct fields
	deepcopier.Copy(conf).To(plugin)

	// Telegraf has a struct wrapper around time.Duration, but it's defined
	// in an internal package which the gocomplier won't compile from
	plugin.CountersRefreshInterval.Duration = time.Duration(conf.CountersRefreshInterval) * time.Millisecond

	// copy nested perf objects
	for _, perfobj := range conf.Object {
		// The perfcounter object is an unexported struct from the original plugin.
		// We can fill this array using anonymous structs.
		plugin.Object = append(plugin.Object, struct {
			ObjectName    string
			Counters      []string
			Instances     []string
			Measurement   string
			WarnOnMissing bool
			FailOnMissing bool
			IncludeTotal  bool
		}{
			perfobj.ObjectName,
			perfobj.Counters,
			perfobj.Instances,
			perfobj.Measurement,
			perfobj.WarnOnMissing,
			perfobj.FailOnMissing,
			perfobj.IncludeTotal,
		})
	}
	return plugin
}

// Configure the monitor and kick off metric syncing
func (m *Monitor) Configure(conf *Config) error {
	plugin := GetPlugin(conf)

	// create the accumulator
	ac := accumulator.NewAccumulator(baseemitter.NewEmitter(m.Output, logger))

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		if err := plugin.Gather(ac); err != nil {
			logger.Error(err)
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

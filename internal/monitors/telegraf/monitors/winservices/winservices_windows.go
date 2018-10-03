// +build windows

package winservices

import (
	"context"
	"time"

	telegrafInputs "github.com/influxdata/telegraf/plugins/inputs"
	telegrafPlugin "github.com/influxdata/telegraf/plugins/inputs/win_services"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/baseemitter"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

// fetch the factory used to generate the perf counter plugin
var factory = telegrafInputs.Inputs["win_services"]

// Configure the monitor and kick off metric syncing
func (m *Monitor) Configure(conf *Config) (err error) {
	plugin := factory().(*telegrafPlugin.WinServices)

	// create the accumulator
	ac := accumulator.NewAccumulator(baseemitter.NewEmitter(m.Output, logger))

	// copy configurations to the plugin
	plugin.ServiceNames = conf.ServiceNames

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		if err := plugin.Gather(ac); err != nil {
			logger.Error(err)
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return err
}

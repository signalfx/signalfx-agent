package exec

import (
	"context"
	"time"

	"github.com/ulule/deepcopier"

	telegrafInputs "github.com/influxdata/telegraf/plugins/inputs"
	telegrafPlugin "github.com/influxdata/telegraf/plugins/inputs/exec"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/pkg/monitors/telegraf/common/emitter/baseemitter"
	"github.com/signalfx/signalfx-agent/pkg/monitors/telegraf/common/parser"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	"github.com/signalfx/signalfx-agent/pkg/utils/timeutil"
	log "github.com/sirupsen/logrus"
)

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"false"`
	Commands             []string          `yaml:"commands"`
	Command              string            `yaml:"command"`
	Timeout              timeutil.Duration `yaml:"timeout"`

	// telegrafParser is a nested object that defines configurations for a Telegraf parser.
	// Please refer to the Telegraf documentation for more information on Telegraf parsers.
	TelegrafParser *parser.Config `yaml:"telegrafParser"`

	// A list of metric names that should be typed as "cumulative counters" in
	// SignalFx.  The Telegraf exec plugin only emits `untyped` metrics, which
	// will by default be sent as SignalFx gauges.
	SignalFxCumulativeMetrics []string `yaml:"signalFxCumulativeMetrics"`
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
	plugin *telegrafPlugin.Exec
}

// fetch the factory used to generate the perf counter plugin
var factory = telegrafInputs.Inputs["exec"]

// Configure the monitor and kick off metric syncing
func (m *Monitor) Configure(conf *Config) (err error) {
	m.plugin = factory().(*telegrafPlugin.Exec)

	cumulativeCounterSet := utils.StringSliceToMap(conf.SignalFxCumulativeMetrics)

	// copy configurations to the plugin
	if err = deepcopier.Copy(conf).To(m.plugin); err != nil {
		logger.Error("unable to copy configurations to plugin")
		return err
	}

	parser, err := conf.TelegrafParser.GetTelegrafParser()
	if err != nil {
		return err
	}

	m.plugin.SetParser(parser)

	emitter := baseemitter.NewEmitter(m.Output, logger)
	emitter.OmitPluginDimension = true

	accumulator := accumulator.NewAccumulator(emitter)

	emitter.SetOmitOriginalMetricType(true)
	emitter.AddDatapointTransformation(func(dp *datapoint.Datapoint) error {
		if cumulativeCounterSet[dp.Metric] {
			dp.MetricType = datapoint.Counter
		}
		if val := dp.Dimensions["signalfx_type"]; val == "cumulative" {
			dp.MetricType = datapoint.Counter
			delete(dp.Dimensions, "signalfx_type")
		}
		return nil
	})

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		if err := m.plugin.Gather(accumulator); err != nil {
			logger.WithError(err).Errorf("an error occurred while gathering metrics")
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return err
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

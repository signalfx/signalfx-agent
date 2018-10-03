package telegraflogparser

import (
	"context"
	telegrafInputs "github.com/influxdata/telegraf/plugins/inputs"
	telegrafPlugin "github.com/influxdata/telegraf/plugins/inputs/logparser"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/baseemitter"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
	"github.com/ulule/deepcopier"
	"time"
)

const monitorType = "telegraf/logparser"

// MONITOR(telegraf/logparser): This monitor is based on the Telegraf logparser plugin.
// The monitor tails log files. More information about the Telegraf plugin
// can be found [here](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/logparser).
// All metrics emitted from this monitor will have the `plugin` dimension set to `telegraf-logparser`
//
// Sample YAML configuration:
//
// ```yaml
//  - type: telegraf/logparser
//    files:
//     - '$file'
//    watchMethod: poll       # specify the file watch method ("inotify" or "poll")
//    fromBeginning: true     # specify to read from the beginning
//    measurementName: test-measurement # the metric name prefix
//    patterns:
//     - "%{COMMON_LOG_FORMAT}" # specifies the apache common log format
//    timezone: UTC
// ```
//

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"false"`
	// Paths to files to be tailed
	Files []string `yaml:"files" validate:"required"`
	// Method for watching changes to files ("ionotify" or "poll")
	WatchMethod string `yaml:"watchMethod" default:"poll"`
	// Whether to start tailing from the beginning of the file
	FromBeginning bool `yaml:"fromBeginning" default:"false"`
	// Name of the measurement
	MeasurementName string `yaml:"measurementName"`
	// A list of patterns to match.
	Patterns []string `yaml:"patterns"`
	// A list of named grok patterns to match.
	NamedPatterns []string `yaml:"namedPatterns"`
	// Custom grok patterns. (`grok` only)
	CustomPatterns string `yaml:"customPatterns"`
	// List of paths to custom grok pattern files.
	CustomPatternFiles []string `yaml:"customPatternFiles"`
	// Specifies the timezone.  The default is UTC time.  Other options are `Local` for the
	// local time on the machine, `UTC`, and `Canada/Eastern` (unix style timezones).
	TimeZone string `yaml:"timezone"`
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel context.CancelFunc
	plugin *telegrafPlugin.LogParserPlugin
}

// fetch the factory function used to generate the plugin
var factory = telegrafInputs.Inputs["logparser"]

// Configure the monitor and kick off metric syncing
func (m *Monitor) Configure(conf *Config) (err error) {
	m.plugin = factory().(*telegrafPlugin.LogParserPlugin)

	// copy configurations to the plugin
	if err = deepcopier.Copy(conf).To(m.plugin); err != nil {
		logger.Error("unable to copy configurations to plugin")
		return err
	}

	grokConf := telegrafPlugin.GrokConfig{}

	// copy configurations to the plugin
	if err = deepcopier.Copy(conf).To(&grokConf); err != nil {
		logger.Error("unable to copy grok configurations to plugin")
		return err
	}

	m.plugin.GrokConfig = grokConf

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// create the emitter
	em := baseemitter.NewEmitter(m.Output, logger)

	// Hard code the plugin name because the emitter will parse out the
	// configured measurement name as plugin and that is confusing.
	em.AddTag("plugin", "telegraf-logparser")

	// create the accumulator
	ac := accumulator.NewAccumulator(em)

	// start the tail plugin
	if err = m.plugin.Start(ac); err != nil {
		return err
	}

	// look for new files to tail on the defined interval
	utils.RunOnInterval(ctx, func() {
		if err := m.plugin.Gather(ac); err != nil {
			logger.Error(err)
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.plugin != nil {
		// stop the collection interval
		m.cancel()
		// stop the telegraf plugin
		m.plugin.Stop()
	}
}

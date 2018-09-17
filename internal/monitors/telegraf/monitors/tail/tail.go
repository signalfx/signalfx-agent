package tail

import (
	telegrafInputs "github.com/influxdata/telegraf/plugins/inputs"
	telegrafPlugin "github.com/influxdata/telegraf/plugins/inputs/tail"
	telegrafParsers "github.com/influxdata/telegraf/plugins/parsers"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/baseemitter"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/parser"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	log "github.com/sirupsen/logrus"
	"github.com/ulule/deepcopier"
)

const monitorType = "telegraf/tail"


// MONITOR(telegraf/tail): This monitor is based on the Telegraf tail plugin.  The monitor tails files and
// named pipes.  The Telegraf parser configured with this monitor extracts metrics in different
// (formats)[https://github.com/influxdata/telegraf/blob/master/docs/DATA_FORMATS_INPUT.md]
// from the tailed output. More information about the Telegraf plugin
// can be found [here](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/tail).
//
// Sample YAML configuration:
//
// ```yaml
// monitors:
//  - type: telegraf/tail
//    files:
//     - '/logs/**.log'       # find all .log files in /logs
//     - '/logs/*/*.log'      # find all .log files who are contained in a directory under /logs/
//     - '/var/log/agent.log' # tail the specified log file
//    watchMethod: inotify    # specify the file watch method ("ionotify" or "poll")
// ```
//
// Sample YAML configuration that specifies a parser:
//
// ```yaml
// monitors:
//  - type: telegraf/tail
//    files:
//     - '/logs/**.log'       # find all .log files in /logs
//     - '/logs/*/*.log'      # find all .log files who are contained in a directory under /logs/
//     - '/var/log/agent.log' # tail the specified log file
//    watchMethod: inotify    # specify the file watch method ("inotify" or "poll")
//    telegrafParser:         # specify a parser
//      dataFormat: "influx"  # set the parser's dataFormat
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
	// Indicates if the file is a named pipe
	Pipe bool `yaml:"pipe" default:"false"`
	// Whether to start tailing from the beginning of the file
	FromBeginning bool `yaml:"fromBeginning" default:"false"`
	// telegrafParser is a nested object that defines configurations for a Telegraf parser.
	// Please refer to the Telegraf (documentation)[https://github.com/influxdata/telegraf/blob/master/docs/DATA_FORMATS_INPUT.md]
	// for more information on Telegraf parsers.
	TelegrafParser *parser.Config `yaml:"telegrafParser"`
	parser telegrafParsers.Parser
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	plugin *telegrafPlugin.Tail
}

// fetch the factory function used to generate the plugin
var factory = telegrafInputs.Inputs["tail"]

// Configure the monitor and kick off metric syncing
func (m *Monitor) Configure(conf *Config) (err error) {
	plugin := factory().(*telegrafPlugin.Tail)

	// use the default config
	if conf.TelegrafParser == nil {
		logger.Debug("defaulting to influx parser because no parser was specified")
		conf.TelegrafParser = &parser.Config{DataFormat: "influx"}
	}

	// set the parser using the provided configuration
	if conf.parser, err = conf.TelegrafParser.GetTelegrafParser(); err != nil {
		return err
	}

	// copy configurations to the plugin
	if err = deepcopier.Copy(conf).To(plugin); err != nil {
		logger.Error("unable to copy configurations to plugin")
		return err
	}

	// set the parser on the plugin
	plugin.SetParser(conf.parser)

	// create the accumulator
	ac := accumulator.NewAccumulator(baseemitter.NewEmitter(m.Output, logger))

	// start the tail plugin
	return plugin.Start(ac)
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	// stop the telegraf plugin
	m.plugin.Stop()
}

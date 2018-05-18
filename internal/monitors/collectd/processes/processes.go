// +build !windows

package processes

//go:generate collectd-template-to-go processes.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/processes"

// MONITOR(collectd/processes): Gathers information about processes running on
// the host.  See
// https://collectd.org/documentation/manpages/collectd.conf.5.shtml#plugin_processes
// and https://collectd.org/wiki/index.php/Plugin:Processes for more
// information on the configuration options.
//
// Example:
//
// ```yaml
//   - type: collectd/processes
//     processes:
//       - mysql
//       - myapp
//     processMatch:
//       docker: "docker.*"
//     collectContextSwitch: true
// ```
//
// The above config will send process metrics for processes named *mysql* and
// *myapp*, along with additional metrics on the number of context switches the
// process has made.  Also, all processes that start with `docker` will have
// their process metrics aggregated together and sent with a `plugin_instance`
// value of `docker`.

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			MonitorCore: *collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `singleInstance:"true"`
	// A list of process names to match
	Processes []string `yaml:"processes"`
	// A map with keys specifying the `plugin_instance` value to be sent for
	// the values which are regexes that match process names.  See example in
	// description.
	ProcessMatch map[string]string `yaml:"processMatch"`
	// Collect metrics on the number of context switches made by the process
	CollectContextSwitch bool `yaml:"collectContextSwitch" default:"false"`
	// The path to the proc filesystem -- useful to override if the agent is
	// running in a container.
	ProcFSPath string `yaml:"procFSPath" default:"/proc"`
}

// Validate will check the config for correctness.
func (c *Config) Validate() error {
	return nil
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) error {
	return am.SetConfigurationAndRun(conf)
}

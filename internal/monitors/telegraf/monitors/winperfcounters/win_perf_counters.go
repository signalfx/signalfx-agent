package winperfcounters

import (
	"time"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	log "github.com/sirupsen/logrus"
)

const monitorType = "telegraf/win_perf_counters"

// MONITOR(telegraf/win_perf_counters): This monitor reads Windows performance
// counters
//
// Sample YAML configuration:
//
// ```yaml
// monitors:
//  - type: telegraf/win_perf_counters
//    printValid: true
//    objects:
//     - objectName: "Processor"
//       instances:
//        - "*"
//       counters:
//        - "% Idle Time"
//        - "% Interrupt Time"
//        - "% Privileged Time"
//        - "% User Time"
//        - "% Processor Time"
//       includeTotal: true
//       measurement: "win_cpu"
// ```
//

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// PerfCounterObj represents a windows performance counter object to monitor
type PerfCounterObj struct {
	// The name of a windows performance counter object
	ObjectName string `yaml:"objectName"`
	// The name of the counters to collect from the performance counter object
	Counters []string `yaml:"counters" default:"[]"`
	// The windows performance counter instances to fetch for the performance counter object
	Instances []string `yaml:"instances" default:"[]"`
	// The name of the telegraf measurement that will be used as a metric name
	Measurement string `yaml:"measurement"`
	// Log a warning if the perf counter object is missing
	WarnOnMissing bool `yaml:"warnOnMissing" default:"false"`
	// Panic if the performance counter object is missing (this will stop the agent)
	FailOnMissing bool `yaml:"failOnMissing" default:"false"`
	// Include the total instance when collecting performance counter metrics
	IncludeTotal bool `yaml:"includeTotal" default:"false"`
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"false" deepcopier:"skip"`
	Object               []PerfCounterObj `yaml:"objects" default:"[]"`
	// Number of nanoseconds that wildcards in counter paths should be expanded
	// and how often to refresh counters from configuration
	CountersRefreshInterval time.Duration `yaml:"counterRefreshInterval" default:"5s"`
	// If `true`, instance indexes will be included in instance names, and wildcards will
	// be expanded and localized (if applicable).  If `false`, non partial wildcards will
	// be expanded and instance names will not include instance indexes.
	UseWildcardsExpansion bool `yaml:"useWildCardExpansion"`
	// Print out the configurations that match available performance counters
	PrintValid bool `yaml:"printValid"`
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
}

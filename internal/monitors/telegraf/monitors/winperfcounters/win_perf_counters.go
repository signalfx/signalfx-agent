package winperfcounters

import (
	"strings"
	"time"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/measurement"
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
	// The frequency that counter paths should be expanded
	// and how often to refresh counters from configuration.
	// This is expressed as a duration.
	CountersRefreshInterval time.Duration `yaml:"counterRefreshInterval" default:"5s"`
	// If `true`, instance indexes will be included in instance names, and wildcards will
	// be expanded and localized (if applicable).  If `false`, non partial wildcards will
	// be expanded and instance names will not include instance indexes.
	UseWildcardsExpansion bool `yaml:"useWildCardExpansion"`
	// Print out the configurations that match available performance counters
	PrintValid bool `yaml:"printValid"`
	// If `true`, metric names will be emitted in the format emitted by the
	// SignalFx PerfCounterReporter
	PCRMetricNames bool `yaml:"pcrMetricNames" default:"false"`
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

// NewPCRReplacer returns a new replacer for sanitizing metricnames and instances like
// SignalFx PCR
func NewPCRReplacer() *strings.Replacer {
	return strings.NewReplacer(
		" ", "_", // PCR bad char
		";", "_", // PCR bad char
		":", "_", // PCR bad char
		"/", "_", // PCR bad char
		"(", "_", // PCR bad char
		")", "_", // PCR bad char
		"*", "_", // PCR bad char
		"\\", "_", // PCR bad char
		"#", "num", // telegraf -> PCR
		"percent", "pct", // telegraf -> PCR
		"_persec", "_sec", // telegraf -> PCR
		"._", "_", // telegraf -> PCR (this is more of a side affect of telegraf's conversion)
		"____", "_", // telegraf -> PCR (this is also a side affect)
		"___", "_", // telegraf -> PCR (this is also a side affect)
		"__", "_") // telegraf/PCR (this is a side affect of both telegraf and PCR conversion)
}

// NewPCRMetricNamesTransformer returns a function for tranforming perf counter
// metric names as parsed from telegraf into something matching the
// SignalFx PerfCounterReporter
func NewPCRMetricNamesTransformer() func(string) string {
	replacer := NewPCRReplacer()
	return func(in string) string {
		return replacer.Replace(strings.ToLower(in))
	}
}

// NewPCRInstanceTagTransformer returns a function for transforming perf counter measurements
func NewPCRInstanceTagTransformer() func(*measurement.Measurement) error {
	replacer := NewPCRReplacer()
	return func(ms *measurement.Measurement) error {
		for t, v := range ms.Tags {
			if t == "instance" {
				v = replacer.Replace(strings.ToLower(v))
				ms.Tags["instance"] = v
			}
		}
		return nil
	}
}

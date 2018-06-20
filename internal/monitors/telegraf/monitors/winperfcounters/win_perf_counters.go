package winperfcounters

import (
	"context"

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

// Perfcounterobj represents a perfcounter object to monitor
type Perfcounterobj struct {
	ObjectName    string   `yaml:"objectName"`
	Counters      []string `yaml:"counters" default:"[]"`
	Instances     []string `yaml:"instances" default:"[]"`
	Measurement   string   `yaml:"measurement"`
	WarnOnMissing bool     `yaml:"warnOnMissing" default:"false"`
	FailOnMissing bool     `yaml:"failOnMissing" default:"false"`
	IncludeTotal  bool     `yaml:"includeTotal" default:"false"`
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"false" deepcopier:"skip"`
	Object               []Perfcounterobj `yaml:"objects" default:"[]"`
	// number of nanoseconds that wildcards in counter paths should be expanded
	// and how often to refresh counters from configuration
	CountersRefreshInterval int `yaml:"counterRefreshInterval" default:"60"`
	// if `true`, instance indexes will be included in instance names, and wildcards will
	// be expanded and localized (if applicable).  If `false`, non partial wildcards will be expanded and instance names will not include instance indexs.
	UseWildcardsExpansion bool
	// print out the configurations that match available performance counters
	PrintValid bool `yaml:"printValid"`
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
	ctx    context.Context
}

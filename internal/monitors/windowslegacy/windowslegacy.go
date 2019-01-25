package windowslegacy

import (
	"time"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	log "github.com/sirupsen/logrus"
)

const monitorType = "windows-legacy"

// MONITOR(windows-legacy):
// This monitor reports metrics for Windows system Performance Counters.
// The metric names are intended to match what was originally reported by
// the SignalFx [PerfCounterReporter](https://github.com/signalfx/PerfCounterReporter)
// The metric types are all gauges as originally reported by the PerfCounterReporter.
//
// ## Windows Performance Counters
// The underlying source for these metrics are Windows Performance Counters.
// Most of the performance counters that we query in this monitor are actually
// rates per second and percentages that we report as instantaneous Gauge values
// each collection interval.
//
// Sample YAML configuration:
//
// ```yaml
// monitors:
//  - type: windows-iis
// ```

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"true" acceptsEndpoints:"false"`
	// (Windows Only) Number of seconds that wildcards in counter paths should
	// be expanded and how often to refresh counters from configuration.
	CountersRefreshInterval time.Duration `yaml:"counterRefreshInterval" default:"60s"`
	// (Windows Only) Print out the configurations that match available
	// performance counters.  This used for debugging.
	PrintValid bool `yaml:"printValid"`
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

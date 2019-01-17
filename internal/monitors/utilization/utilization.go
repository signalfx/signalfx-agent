package utilization

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	log "github.com/sirupsen/logrus"
)

const monitorType = "signalfx-system-utilization"

// MONITOR(signalfx-system-utilization):
// This monitor reports utilization metrics for Windows
// Hosts.  It is used to drive the Windows Smart Agent host navigator view and dashboard
// content.
//
// Sample YAML configuration:
//
// ```yaml
// monitors:
//  - type: signalfx-system-utilization
// ```

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"false"`
	// (Windows Only) number of seconds that wildcards in counter paths should
	// be expanded and how often to refresh counters from configuration.
	CountersRefreshInterval int `yaml:"counterRefreshInterval" default:"300000"`
	// (Windows Only) print out the configurations that match available
	// performance counters.  This used for debugging.
	PrintValid bool `yaml:"printValid"`
}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
}

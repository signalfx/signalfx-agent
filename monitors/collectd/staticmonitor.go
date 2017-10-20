package collectd

import (
	"text/template"

	"github.com/signalfx/neo-agent/core/config"
)

// StaticMonitorCore is intended to be embedded in the individual monitors that
// represent static collectd plugins that don't depend on service discovery at
// all (e.g. the metadata plugin).  It does very little over BaseMonitor but is
// here for completeness sake.
type StaticMonitorCore struct {
	BaseMonitor
}

// NewStaticMonitorCore creates a new un-configured instance
func NewStaticMonitorCore(template *template.Template) *StaticMonitorCore {
	return &StaticMonitorCore{
		BaseMonitor: *NewBaseMonitor(template),
	}
}

// SetConfigurationAndRun sets the configuration to be used when rendering
// templates, and writes config before queueing a collectd restart.
func (smc *StaticMonitorCore) SetConfigurationAndRun(conf config.MonitorCustomConfig) bool {
	if !smc.SetConfiguration(conf) {
		return false
	}
	return smc.WriteConfigForPluginAndRestart()
}
